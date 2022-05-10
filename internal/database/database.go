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
	"strconv"
	"time"
)

type GetWalletData struct {
	current    string
	withdrawed string
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
								current_value VARCHAR,
								withdrawed VARCHAR
					);`
	res3, err3 := db.ExecContext(ctx, sqlCreateWalletDB)
	sqlCreateWithdrawsDB := `CREATE TABLE IF NOT EXISTS withdraws (
								id serial PRIMARY KEY,
								sum_withdrawed integer ,
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

func (db *PGDataBase) UpdateWallet(order string, value float32, ctx context.Context) error {
	log.Println("Start Update Wallet order:", order)

	log.Println("Find owner of order:")
	sqlGetUser := `SELECT owner FROM orders WHERE order_temp=$1;`
	user := db.conn.QueryRowContext(ctx, sqlGetUser, order)
	result := GetUserData{}

	if err := user.Scan(&result.login); err != nil {
		log.Fatal(err)
	}

	log.Println("Owner is::", &result.login)
	log.Println("Find current value:")
	sqlGetWallet := `SELECT current_value FROM wallet WHERE owner=$1;`
	wallet := db.conn.QueryRowContext(ctx, sqlGetWallet, &result.login)
	result2 := GetWalletData{}

	if err := wallet.Scan(&result2.current); err != nil {
		log.Fatal(err)
	}

	sqlSetStatus := `UPDATE wallet SET current_value = ($1) WHERE owner = ($2);`
	f, err := strconv.ParseFloat(result2.current, 32)
	s := fmt.Sprintf("%f", float32(f)+value)
	log.Println("s: ", s)
	log.Println("Current value: ", s)
	_, err = db.conn.QueryContext(ctx, sqlSetStatus, s, order)
	if err != nil {
		log.Println("err db.conn.QueryContext(ctx, sqlSetStatus, status, order)", err)
		return err
	}
	return nil
}

func (db *PGDataBase) UpdateStatus(order string, status string, ctx context.Context) error {
	log.Println("Start UpdateStatus order:", order)
	sqlSetStatus := `UPDATE orders SET status = ($1) WHERE order_temp = ($2);`
	_, err := db.conn.QueryContext(ctx, sqlSetStatus, status, order)
	if err != nil {
		log.Println("err UpdateStatus", err)
		return err
	}
	log.Println("Good End UpdateStatus")
	return nil
}
func (db *PGDataBase) UpdateAccural(order string, accural string, ctx context.Context) error {
	log.Println("Start UpdateStatus order:", order)
	sqlSetAccural := `UPDATE orders SET accural = ($1) WHERE order_temp = ($2);`
	_, err := db.conn.QueryContext(ctx, sqlSetAccural, accural, order)
	if err != nil {
		log.Println("err UpdateAccural", err)
		return err
	}
	log.Println("Good End UpdateAccural")
	return nil
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
	//sqlSetAuth := `UPDATE users SET is_authed = true WHERE login = ANY ($1);`
	//queryauth := db.conn.QueryRowContext(ctx, sqlSetAuth, login)
	//queryauth.Scan(&result.authed)
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
		log.Println(err)
	}
	sqlAddWallet := `INSERT INTO wallet (owner, current_value, withdrawed)
				  VALUES ($1, $2, $3)`

	_, err = db.conn.ExecContext(ctx, sqlAddWallet, login, "0", "0")
	if err != nil {
		log.Println("err db.conn.180 Reg ", err)
		return err
	}
	log.Println("err is nil")
	return err
}
func (db *PGDataBase) UploadOrder(login string, order string, ctx context.Context) error {
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

type RespOrder struct {
	Order      string
	Status     string
	Accrual    string
	UploadedAt time.Time
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
		var u RespOrder
		err = rows.Scan(&u.Order, &u.Status, &u.Accrual, &u.UploadedAt)
		if err != nil {
			log.Println("err  rows.Scan(&u.Order, &u.Status, &u.Accrual, &u.UploadedAt)")
			return result, handlers.NewErrorWithDB(errors.New("GetOrder"), "GetOrder")
		}
		//result = append(result, u)
		intAccrual, err := strconv.ParseFloat(u.Accrual, 32)
		if err != nil {
			log.Println("err  Atoi")
			return result, err
		}
		if u.Accrual != "0" {
			result = append(result, handlers.ResponseOrder{
				Order:      u.Order,
				Status:     u.Status,
				Accrual:    int(intAccrual),
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

	query := db.conn.QueryRowContext(ctx, sqlGetBalance, login)

	//Тут может быть проблема конфертации
	err := query.Scan(&result.Current, &result.Withdrawn)

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
func (db *PGDataBase) Withdraw(login string, order string, value float64, ctx context.Context) error {
	db.UploadOrder(login, order, ctx)

	sqlGetWallet := `SELECT current_value, withdrawed FROM wallet WHERE owner=$1;`
	result := GetWalletData{}
	query := db.conn.QueryRowContext(ctx, sqlGetWallet, login)
	err := query.Scan(&result.current, &result.withdrawed) //or how check empty value?
	f, err := strconv.ParseFloat(result.current, 64)
	if value > f {
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
	current := f - value

	f2, err := strconv.ParseFloat(result.withdrawed, 64)
	withdrawed := fmt.Sprintf("%f", f2+value)
	log.Println("withdrawed: ", withdrawed)
	sqlUpdateWallet := `UPDATE wallet SET current_value = ($1), withdrawed = ($2) WHERE owner = ($3);`
	s := fmt.Sprintf("%f", current)
	_, err = db.conn.QueryContext(ctx, sqlUpdateWallet, s, withdrawed, login)

	if err != nil {
		log.Println("err sqlUpdateWallet")
		return err
	}
	//402 — на счету недостаточно средств;
	//422 — неверный номер заказа;
	return nil
}

type OrderInfoString struct {
	Order   string
	Status  string
	Accrual string
}

func (db *PGDataBase) GetOrderInfo(order string, ctx context.Context) (handlers.ResponseOrderInfo, error) {
	result := handlers.ResponseOrderInfo{}

	sqlGetOrder := `SELECT order_temp, status, accural FROM orders WHERE order_temp=($1);`
	rows := db.conn.QueryRowContext(ctx, sqlGetOrder, order)

	if rows.Err() != nil {
		log.Println("err rows.Err() != nil")
		return result, rows.Err()
	}
	var u OrderInfoString //handlers.ResponseOrderInfo
	err := rows.Scan(&u.Order, &u.Status, &u.Accrual)
	if err != nil {
		log.Println("err  rows.Scan(&u.Order, &u.Status, &u.Accrual, &u.UploadedAt)")
		return result, err
	}
	//result = append(result, u)
	intAccrual, err := strconv.Atoi(u.Accrual)
	result = handlers.ResponseOrderInfo{
		Order:   u.Order,
		Status:  u.Status,
		Accrual: intAccrual,
	}

	return result, nil
}

type ResponseWithdrawsLocal struct {
	Order       string
	Sum         string
	ProcessedAt time.Time
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
		var u ResponseWithdrawsLocal
		err = rows.Scan(&u.Order, &u.Sum, &u.ProcessedAt)
		if err != nil {
			log.Println("err  rows.Scan(&u.Order, &u.Sum,&u.ProcessedAt)")
			return result, handlers.NewErrorWithDB(errors.New("ResponseWithdraws"), "ResponseWithdraws")
		}
		//intAccrual, err := strconv.ParseFloat(u.Accrual,32)
		intSum, err := strconv.ParseFloat(u.Sum, 32)

		if err != nil {
			log.Println("err  Atoi")
			return result, err
		}
		result = append(result, handlers.ResponseWithdraws{
			Order:       u.Order,
			Sum:         int(intSum),
			ProcessedAt: u.ProcessedAt,
		})

	}
	return result, nil
}
func (db *PGDataBase) GetNewOrder(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var order string
	sqlGetNewOrder := `SELECT order_temp FROM orders WHERE status in('PROCESSING', 'NEW') ORDER BY random() LIMIT 1;`
	row := db.conn.QueryRowContext(ctx, sqlGetNewOrder)
	row.Scan(&order)

	return order, nil
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
