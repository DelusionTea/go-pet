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
	"time"
)

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
								order VARCHAR NOT NULL,
								order_id uuid DEFAULT uuid_generate_v4 (), 	
								status VARCHAR NOT NULL DEFAULT 'NEW', 
								accurual VARCHAR,
								uploaded_at DATE NOT NULL DEFAULT CURRENT_DATE
					);`
	res2, err2 := db.ExecContext(ctx, sqlCreateOrdersDB)
	log.Println("Create second TABLE")
	//sqlCreateDB := `CREATE TABLE IF NOT EXISTS urls (
	//							id serial PRIMARY KEY,
	//							user_id uuid DEFAULT uuid_generate_v4 (),
	//							origin_url VARCHAR NOT NULL,
	//							short_url VARCHAR NOT NULL UNIQUE,
	//							is_deleted BOOLEAN NOT NULL DEFAULT FALSE
	//				);`
	//res, err := db.ExecContext(ctx, sqlCreateDB)
	log.Println("Create table", "1.", err1, res1, "2", err2, res2)
	return nil
}

type PGDataBase struct {
	conn *sql.DB
}

type OrderInfo struct {
	Order      string    `json:"order"`
	Status     string    `json:"status"`
	Accrual    int       `json:"accrual"`
	UploadedAt time.Time `json:"uploaded_at"`
}
type GetUserData struct {
	login    string
	password string
	authed   bool
}

func (db *PGDataBase) UpdateStatus(status string, ctx context.Context) {
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
	log.Println("err is nil")
	return err
}
func (db *PGDataBase) UploadOrder(login string, order []byte, ctx context.Context) error {
	//Вначале селект. Если не пусто, то делаем проверку - если там другой пользак то alredy here, если другой то Conflict
	sqlCheckOrder := `SELECT owner FROM orders WHERE order=$1 FETCH FIRST ROW ONLY;`

	result := GetUserData{}
	query := db.conn.QueryRowContext(ctx, sqlCheckOrder, order)
	err := query.Scan(&result.login) //or how check empty value?
	if result.login != "" {
		log.Println("DB ERROR OF UPLOAD ORDER")
		log.Println(result.login)
		log.Println(login)
		if result.login == login {
			log.Println("Alredy here")
			return handlers.NewErrorWithDB(errors.New("Alredy here"), "Alredy here")
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
	}

	sqlAddUser := `INSERT INTO users (login, order)
				  VALUES ($1, $2)`

	_, err = db.conn.ExecContext(ctx, sqlAddUser, login, order)

	return err
}
func (db *PGDataBase) GetOrder(status string, ctx context.Context) {
	return
}
func (db *PGDataBase) GetBalance(status string, ctx context.Context) {
	return
}
func (db *PGDataBase) Withdraw(status string, ctx context.Context) {
	return
}
func (db *PGDataBase) GetWithdraws(status string, ctx context.Context) {
	return
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
