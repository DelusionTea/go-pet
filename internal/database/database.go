package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/DelusionTea/go-pet.git/internal/app/handlers"
	"github.com/jackc/pgerrcode"
	"github.com/lib/pq"
	"log"
)

type GetWalletData struct {
	current    float64
	withdrawed int
}

func NewDatabaseRepository(db *sql.DB) handlers.MarketInterface {
	return handlers.MarketInterface(NewDatabase(db))
}

func SetUpDataBase(db *sql.DB, ctx context.Context) error {

	var extention string
	query := db.QueryRowContext(ctx, "SELECT 'exists' FROM pg_extension WHERE extname='uuid-ossp';")
	query.Scan(&extention)
	if extention != "exists" {
		_, err := db.ExecContext(ctx, `CREATE EXTENSION "uuid-ossp";`)
		if err != nil {
			return err
		}
		log.Println("Create EXTENSION")
	}
	sqlCreateUsersDB := `CREATE TABLE IF NOT EXISTS users (
								id serial PRIMARY KEY,
								user_id uuid DEFAULT uuid_generate_v4 (), 	
								login VARCHAR NOT NULL UNIQUE, 
								password VARCHAR NOT NULL UNIQUE,
								is_authed BOOLEAN NOT NULL DEFAULT FALSE
					);`
	log.Println("Create first TABLE")
	res1, err1 := db.ExecContext(ctx, sqlCreateUsersDB)
	sqlCreateOrdersDB := `CREATE TABLE IF NOT EXISTS orders (
								id serial PRIMARY KEY,
								owner VARCHAR NOT NULL,
								order_temp VARCHAR NOT NULL,
								order_id uuid DEFAULT uuid_generate_v4 (), 	
								status VARCHAR, 
								accural VARCHAR DEFAULT '0',
								uploaded_at DATE NOT NULL DEFAULT CURRENT_DATE
					);`
	//SELECT order_temp, status, accural, uploadet_at FROM orders WHERE owner=$1
	res2, err2 := db.ExecContext(ctx, sqlCreateOrdersDB)
	log.Println("Create second TABLE")
	sqlCreateWalletDB := `CREATE TABLE IF NOT EXISTS wallet (
								id serial PRIMARY KEY,
								owner VARCHAR NOT NULL UNIQUE,
								current_value double precision,
								withdrawed integer
					);`
	res3, err3 := db.ExecContext(ctx, sqlCreateWalletDB)
	sqlCreateWithdrawsDB := `CREATE TABLE IF NOT EXISTS withdraws (
								id serial PRIMARY KEY,
								sum_withdrawed integer NOT NULL,
								order_temp VARCHAR NOT NULL UNIQUE,
								owner VARCHAR NOT NULL UNIQUE,
								uploaded_at DATE NOT NULL DEFAULT CURRENT_DATE
					);`
	res4, err4 := db.ExecContext(ctx, sqlCreateWithdrawsDB)
	log.Println("Create table", "1.", err1, res1, "2", err2, res2, "3", err3, res3, "4", err4, res4)
	return nil
}

type PGDataBase struct {
	conn *sql.DB
}

type GetUserData struct {
	login    string
	password string
	authed   bool
}

func (db *PGDataBase) UpdateWallet(order string, value float64, ctx context.Context) error {
	sqlGetUser := `SELECT owner FROM orders WHERE order_temp=$1 FETCH FIRST ROW ONLY;`
	user, err := db.conn.QueryContext(ctx, sqlGetUser, order)
	if err != nil {
		log.Println("err db.conn.QueryContext(ctx, sqlGetUser, status, order)")
		return err
	}
	result := GetUserData{}
	user.Scan(&result.login)

	sqlGetWallet := `SELECT current_value FROM wallet WHERE owner=$1 FETCH FIRST ROW ONLY;`
	wallet, err := db.conn.QueryContext(ctx, sqlGetWallet, &result.login)
	if err != nil {
		log.Println("err db.conn.QueryContext(ctx, sqlGetUser, status, order)")
		return err
	}

	result2 := GetWalletData{}
	wallet.Scan(&result2.current)

	sqlSetStatus := `UPDATE wallet SET current_value = ($1) WHERE owner = ANY ($2);`
	_, err = db.conn.QueryContext(ctx, sqlSetStatus, result2.current+value, order)
	if err != nil {
		log.Println("err db.conn.QueryContext(ctx, sqlSetStatus, status, order)")
		return err
	}
	return nil
}

func (db *PGDataBase) UpdateStatus(order string, status string, ctx context.Context) {
	sqlSetStatus := `UPDATE orders SET status = ($1) WHERE order_temp = ANY ($2);`
	_, err := db.conn.QueryContext(ctx, sqlSetStatus, status, order)
	if err != nil {
		log.Println("err db.conn.QueryContext")
		return
	}
	return
}
func (db *PGDataBase) Login(login string, pass string, ctx context.Context) (string, error) {
	log.Println("Start Login")
	sqlGetUser := `SELECT login,password FROM users WHERE login=$1 FETCH FIRST ROW ONLY;`
	query := db.conn.QueryRowContext(ctx, sqlGetUser, login)
	result := GetUserData{}
	query.Scan(&result.login, &result.password)
	if result.login == "" {
		return "", handlers.NewErrorWithDB(errors.New("not found"), "user not found")
	}
	if result.password != pass {
		return "", handlers.NewErrorWithDB(errors.New("wrong password"), "wrong password")
	}
	//UPDATE users SET is_authed = true WHERE login = ANY ($1);
	sqlSetAuth := `UPDATE users SET is_authed = true WHERE login = ANY ($1);`
	queryauth := db.conn.QueryRowContext(ctx, sqlSetAuth, login)
	queryauth.Scan(&result.authed)
	return "Login success", nil
}

func (db *PGDataBase) CheckAuth(login string, ctx context.Context) (string, error) {
	log.Println("Check Auth Start")
	sqlGetStatus := `SELECT login,is_authed FROM users WHERE login=$1 FETCH FIRST ROW ONLY;`
	result := GetUserData{}
	query := db.conn.QueryRowContext(ctx, sqlGetStatus, login)
	query.Scan(&result.login, &result.authed)
	if result.login == "" {
		return "", handlers.NewErrorWithDB(errors.New("not found"), "user not found")
	}
	if result.authed == false {
		return "", handlers.NewErrorWithDB(errors.New("not authed"), "user not authed")
	}
	return "ok", nil
}

func (db *PGDataBase) Register(login string, pass string, ctx context.Context) error {
	log.Println("Start Register")
	sqlAddUser := `INSERT INTO users (login, password)
				  VALUES ($1, $2)`

	_, err := db.conn.ExecContext(ctx, sqlAddUser, login, pass)

	if err, ok := err.(*pq.Error); ok {
		log.Println("DB REGISTER ERROR HERE")
		if err.Code == pgerrcode.UniqueViolation {
			log.Println("UniqueViolation")
			return handlers.NewErrorWithDB(err, "Conflict")
		}
		//if err.Code == pgerrcode.UniqueViolation {
		//	return err //"StatusConflict"
		//	//ctx.IndentedJSON(http.StatusConflict)
		//}
		log.Println(err)
	}
	sqlAddWallet := `INSERT INTO wallet (owner, current_value, withdrawed)
				  VALUES ($1, $2, $3)`

	_, err = db.conn.ExecContext(ctx, sqlAddWallet, login, 0, 0)
	log.Println("err is nil")
	return err
}
func (db *PGDataBase) UploadOrder(login string, order []byte, ctx context.Context) error {
	//Вначале селект. Если не пусто, то делаем проверку - если там другой пользак то alredy here, если другой то Conflict
	sqlCheckOrder := `SELECT owner FROM orders WHERE order_temp=$1 FETCH FIRST ROW ONLY;`

	result := GetUserData{}
	query := db.conn.QueryRowContext(ctx, sqlCheckOrder, order)
	err := query.Scan(&result.login) //or how check empty value?
	if result.login != "" {
		log.Println("DB ERROR OF UPLOAD ORDER")
		log.Println(result.login)
		log.Println(login)
		if result.login == login {
			log.Println("Alredy here")
			return handlers.NewErrorWithDB(errors.New("Alredy here"), "Already here")
		}
		if result.login != login {
			log.Println("Conflict")
			return handlers.NewErrorWithDB(errors.New("Conflict"), "Conflict")
		}

	}

	if err, ok := err.(*pq.Error); ok {
		if err.Code == pgerrcode.NoData {
			log.Println("pgerrcode.NoData")

		}
		if err.Code == pgerrcode.SuccessfulCompletion {
			log.Println("pgerrcode.SuccessfulCompletion")
		}
		if err.Code == pgerrcode.CaseNotFound {
			log.Println("pgerrcode.CaseNotFound")
		}

		log.Println(err)
		return err
	}

	sqlAddOrder := `INSERT INTO orders (owner, order_temp,status)
				  VALUES ($1, $2, $3)`

	_, err = db.conn.ExecContext(ctx, sqlAddOrder, login, order, "NEW")

	return err
}
func (db *PGDataBase) GetOrder(login string, ctx context.Context) ([]handlers.ResponseOrder, error) {

	result := []handlers.ResponseOrder{}

	sqlGetOrder := `SELECT order_temp, status, accural, uploaded_at FROM orders WHERE owner=$1;`
	rows, err := db.conn.QueryContext(ctx, sqlGetOrder, login)
	if err != nil {
		log.Println("err db.conn.QueryContext")
		return result, err
	}
	if rows.Err() != nil {
		log.Println("err rows.Err() != nil")
		return result, rows.Err()
	}
	defer rows.Close()

	for rows.Next() {
		var u handlers.ResponseOrder
		err = rows.Scan(&u.Order, &u.Status, &u.Accrual, &u.UploadedAt)
		if err != nil {
			log.Println("err  rows.Scan(&u.Order, &u.Status, &u.Accrual, &u.UploadedAt)")
			return result, err
		}
		//result = append(result, u)
		if u.Accrual != 0 {
			result = append(result, handlers.ResponseOrder{
				Order:      u.Order,
				Status:     u.Status,
				Accrual:    u.Accrual,
				UploadedAt: u.UploadedAt,
			})
		} else {
			result = append(result, handlers.ResponseOrder{
				Order:      u.Order,
				Status:     u.Status,
				UploadedAt: u.UploadedAt,
			})
		}

	}
	return result, nil
}
func (db *PGDataBase) GetBalance(login string, ctx context.Context) (handlers.BalanceResponse, error) {
	result := handlers.BalanceResponse{}
	sqlGetBalance := `SELECT current_value, withdrawed FROM wallet WHERE owner=$1;`

	query, err := db.conn.QueryContext(ctx, sqlGetBalance, login)
	if err != nil {
		log.Println("err db.conn.QueryContext")
		return result, err
	}
	err = query.Scan(&result.Current, &result.Withdrawn)

	//if result.Current == "" {
	//	return "", handlers.NewErrorWithDB(errors.New("not found"), "user not found")
	//}
	if err != nil {
		log.Println("empty")
		return result, handlers.NewErrorWithDB(errors.New("empty"), "empty")
	}
	result = handlers.BalanceResponse{
		Current:   result.Current,
		Withdrawn: result.Withdrawn,
	}

	return result, nil
}
func (db *PGDataBase) Withdraw(login string, order []byte, value int, ctx context.Context) error {
	db.UploadOrder(login, order, ctx)

	sqlGetWallet := `SELECT current_value, withdrawed FROM withdraws WHERE owner=$1;`
	result := GetWalletData{}
	query := db.conn.QueryRowContext(ctx, sqlGetWallet, login)
	err := query.Scan(&result.current, &result.withdrawed) //or how check empty value?
	if value > int(result.current) {
		//402 — на счету недостаточно средств;
		return handlers.NewErrorWithDB(errors.New("402"), "402")
	}
	//add this to withdraws.
	sqlAddWithdraws := `INSERT INTO withdraws (sum_withdrawed, order_temp, owner)
//				  VALUES ($1, $2, $3)`
	_, err = db.conn.QueryContext(ctx, sqlAddWithdraws, value, order, login)
	if err != nil {
		log.Println("err sqlAddWithdraws")
		return err
	}
	//increase wallet
	current := result.current - float64(value)
	withdrawed := result.withdrawed + value

	sqlUpdateWallet := `UPDATE wallet SET current_value = ($1), withdrawed = ($2) WHERE owner = ANY ($3);`
	_, err = db.conn.QueryContext(ctx, sqlUpdateWallet, current, withdrawed, login)

	if err != nil {
		log.Println("err sqlUpdateWallet")
		return err
	}
	//402 — на счету недостаточно средств;
	//422 — неверный номер заказа;
	return nil
}
func (db *PGDataBase) GetWithdraws(login string, ctx context.Context) ([]handlers.ResponseWithdraws, error) {
	result := []handlers.ResponseWithdraws{}

	sqlGetWithdraw := `SELECT order_temp, sum_withdrawed, uploaded_at FROM withdraws WHERE owner=$1;`
	rows, err := db.conn.QueryContext(ctx, sqlGetWithdraw, login)
	if err != nil {
		log.Println("err db.conn.QueryContext")
		return result, err
	}
	if rows.Err() != nil {
		log.Println("err rows.Err() != nil")
		return result, rows.Err()
	}
	defer rows.Close()

	for rows.Next() {
		var u handlers.ResponseWithdraws
		err = rows.Scan(&u.Order, &u.Sum, &u.ProcessedAt)
		if err != nil {
			log.Println("err  rows.Scan(&u.Order, &u.Sum,&u.ProcessedAt)")
			return result, err
		}

		result = append(result, handlers.ResponseWithdraws{
			Order:       u.Order,
			Sum:         u.Sum,
			ProcessedAt: u.ProcessedAt,
		})

	}
	return result, nil
}

func NewDatabase(db *sql.DB) *PGDataBase {
	result := &PGDataBase{
		conn: db,
	}
	return result
}

func (db *PGDataBase) Ping(ctx context.Context) error {

	err := db.conn.PingContext(ctx)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

//func NewDatabaseRepository(baseURL string, db *sql.DB) handlers.ShorterInterface {
//	return handlers.ShorterInterface(NewDatabase(baseURL, db))
//}

//func (db *PGDataBase) AddURL(ctx context.Context, longURL string, shortURL string, user string) error {
//
//	sqlAddRow := `INSERT INTO urls (user_id, origin_url, short_url)
//				  VALUES ($1, $2, $3)`
//
//	_, err := db.conn.ExecContext(ctx, sqlAddRow, user, longURL, shortURL)
//
//	if err, ok := err.(*pq.Error); ok {
//		//if err.Code == pgerrcode.UniqueViolation {
//		//	return handlers.NewErrorWithDB(err, "UniqConstraint")
//		//}
//		log.Println(err)
//	}
//
//	return err
//}

//func (db *PGDataBase) GetURL(ctx context.Context, shortURL string) (string, error) {
//
//	sqlGetURLRow := `SELECT origin_url, is_deleted FROM urls WHERE short_url=$1 FETCH FIRST ROW ONLY;`
//	query := db.conn.QueryRowContext(ctx, sqlGetURLRow, shortURL)
//	result := GetURLdata{}
//	query.Scan(&result.OriginURL, &result.IsDeleted)
//	//if result.OriginURL == "" {
//	//	return "", handlers.NewErrorWithDB(errors.New("not found"), "Not found")
//	//}
//	//if result.IsDeleted {
//	//	return "", handlers.NewErrorWithDB(errors.New("Deleted"), "Deleted")
//	//}
//	return result.OriginURL, nil
//}

//func (db *PGDataBase) GetUserURL(ctx context.Context, user string) ([]handlers.ResponseGetURL, error) {
//
//	result := []handlers.ResponseGetURL{}
//
//	sqlGetUserURL := `SELECT origin_url, short_url FROM urls WHERE user_id=$1 AND is_deleted=false;`
//	rows, err := db.conn.QueryContext(ctx, sqlGetUserURL, user)
//	if err != nil {
//		return result, err
//	}
//	if rows.Err() != nil {
//		return result, rows.Err()
//	}
//	defer rows.Close()
//
//	for rows.Next() {
//		var u handlers.ResponseGetURL
//		err = rows.Scan(&u.OriginalURL, &u.ShortURL)
//		if err != nil {
//			return result, err
//		}
//		u.ShortURL = db.baseURL + u.ShortURL
//		result = append(result, u)
//	}
//
//	return result, nil
//}

//func (db *PGDataBase) AddURLs(ctx context.Context, urls []handlers.ManyPostURL, user string) ([]handlers.ManyPostResponse, error) {
//
//	result := []handlers.ManyPostResponse{}
//	tx, err := db.conn.Begin()
//
//	if err != nil {
//		return nil, err
//	}
//
//	defer tx.Rollback()
//
//	stmt, err := tx.PrepareContext(ctx, `INSERT INTO urls (user_id, origin_url, short_url) VALUES ($1, $2, $3)`)
//
//	if err != nil {
//		return nil, err
//	}
//
//	defer stmt.Close()
//
//	for _, u := range urls {
//		shortURL := shorter.Shorter(u.OriginalURL)
//		if _, err = stmt.ExecContext(ctx, user, u.OriginalURL, shortURL); err != nil {
//			return nil, err
//		}
//		result = append(result, handlers.ManyPostResponse{
//			CorrelationID: u.CorrelationID,
//			ShortURL:      db.baseURL + shortURL,
//		})
//	}
//
//	if err != nil {
//		return nil, err
//	}
//	tx.Commit()
//	return result, nil
//}

//func (db *PGDataBase) DeleteManyURL(ctx context.Context, urls []string, user string) error {
//
//	sql := `UPDATE urls SET is_deleted = true WHERE short_url = ANY ($1);`
//	urlsToDelete := []string{}
//	for _, url := range urls {
//		if db.isOwner(ctx, url, user) {
//			urlsToDelete = append(urlsToDelete, url)
//		}
//	}
//	_, err := db.conn.ExecContext(ctx, sql, pq.Array(urlsToDelete))
//	if err != nil {
//		return err
//	}
//	return nil
//}
//
//func (db *PGDataBase) isOwner(ctx context.Context, url string, user string) bool {
//	sqlGetURLRow := `SELECT user_id FROM urls WHERE short_url=$1 FETCH FIRST ROW ONLY;`
//	query := db.conn.QueryRowContext(ctx, sqlGetURLRow, url)
//	result := ""
//	query.Scan(&result)
//	return result == user
//}

//func (db *PGDataBase) DeleteURLs(ctx context.Context, urls []string, user string) error {
//	sql := `UPDATE urls SET is_deleted = true WHERE short_url = ANY ($1);`
//	urlsToDelete := []string{}
//	for _, url := range urls {
//		if db.isOwner(ctx, url, user) {
//			urlsToDelete = append(urlsToDelete, url)
//		}
//	}
//	_, err := db.conn.ExecContext(ctx, sql, pq.Array(urlsToDelete))
//	if err != nil {
//		return err
//	}
//	return nil
//}
