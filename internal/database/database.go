package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/DelusionTea/go-pet.git/internal/app/handlers"
	"github.com/DelusionTea/go-pet.git/internal/database/models"
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
func NewDatabase(db *sql.DB) *PGDataBase {
	result := &PGDataBase{
		conn: db,
	}
	return result
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
								sum_withdrawed VARCHAR ,
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

type RespOrder struct {
	Order      string
	Status     string
	Accrual    string
	UploadedAt time.Time
}

type OrderInfoString struct {
	Order   string
	Status  string
	Accrual string
}

type ResponseWithdrawsLocal struct {
	Order       string
	Sum         string
	ProcessedAt time.Time
}

func (db *PGDataBase) UpdateWallet(ctx context.Context, order string, value float32) error {
	// Get a Tx for making transaction requests.
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	log.Println("Start Update Wallet order:", order, "value:", value)

	log.Println("Find owner of order:")
	sqlGetUser := `SELECT owner FROM orders WHERE order_temp=$1;`
	user := tx.QueryRowContext(ctx, sqlGetUser, order)
	result := GetUserData{}

	if err := user.Scan(&result.login); err != nil {
		log.Fatal(err)
	}

	log.Println("Owner is::", &result.login)
	log.Println("Find current value")
	sqlGetWallet := `SELECT current_value FROM wallet WHERE owner=$1;`
	wallet := tx.QueryRowContext(ctx, sqlGetWallet, &result.login)
	result2 := GetWalletData{}

	if err := wallet.Scan(&result2.current); err != nil {
		log.Fatal(err)
	}

	sqlSetStatus := `UPDATE wallet SET current_value = ($1) WHERE owner = ($2);`
	f, err := strconv.ParseFloat(result2.current, 32)
	if err != nil {
		log.Println("err db.conn.QueryContext(ctx, sqlSetStatus, status, order)", err)
		return err
	}
	s := fmt.Sprintf("%f", float32(f)+value)
	log.Println("s: ", s)
	log.Println("Current value: ", s)
	_, err = tx.ExecContext(ctx, sqlSetStatus, s, &result.login)
	if err != nil {
		log.Println("err db.conn.QueryContext(ctx, sqlSetStatus, status, order)", err)
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}
func (db *PGDataBase) UpdateStatus(ctx context.Context, order string, status string) error {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	log.Println("Start UpdateStatus order:", order, " status: ", status)
	sqlSetStatus := `UPDATE orders SET status = ($1) WHERE order_temp = ($2);`
	_, err = tx.ExecContext(ctx, sqlSetStatus, status, order)
	if err != nil {
		log.Println("err UpdateStatus", err)
		return err
	}
	log.Println("Good End UpdateStatus")
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}
func (db *PGDataBase) UpdateAccural(ctx context.Context, order string, accural float32) error {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	log.Println("Start UpdateAccrual order:", order, " accrual: ", accural)
	s := fmt.Sprintf("%f", float32(accural))

	sqlSetAccural := `UPDATE orders SET accural = ($1) WHERE order_temp = ($2);`
	_, err = tx.ExecContext(ctx, sqlSetAccural, s, order)
	if err != nil {
		log.Println("err UpdateAccural", err)
		return err
	}
	log.Println("Good End UpdateAccural")
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}
func (db *PGDataBase) Login(ctx context.Context, login string, pass string) (string, error) {
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
func (db *PGDataBase) Register(ctx context.Context, login string, pass string) error {
	log.Println("Start Register")
	sqlAddUser := `INSERT INTO users (login, password)
				  VALUES ($1, $2)`

	_, err := db.conn.ExecContext(ctx, sqlAddUser, login, pass)

	if err, ok := err.(*pq.Error); ok {
		log.Println("DB REGISTER ERROR HERE")
		if err.Code == pgerrcode.UniqueViolation {
			log.Println("UniqueViolation")
			return handlers.NewErrorWithDB(err, "conflict")
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
func (db *PGDataBase) UploadOrder(ctx context.Context, login string, order string) error {
	sqlCheckOrder := `SELECT owner FROM orders WHERE order_temp=$1 FETCH FIRST ROW ONLY;`

	result := GetUserData{}
	query := db.conn.QueryRowContext(ctx, sqlCheckOrder, order)
	err := query.Scan(&result.login) //or how check empty value?
	if err != nil {
		log.Println(err)
	}
	if result.login != "" {
		log.Println("DB ERROR OF UPLOAD ORDER")
		log.Println(result.login)
		log.Println(login)
		if result.login == login {
			log.Println("Alredy here")
			return handlers.NewErrorWithDB(errors.New("already here"), "already here")
		}
		if result.login != login {
			log.Println("Conflict")
			return handlers.NewErrorWithDB(errors.New("conflict"), "conflict")
		}

	}

	sqlAddOrder := `INSERT INTO orders (owner, order_temp,status)
				  VALUES ($1, $2, $3)`

	_, err = db.conn.ExecContext(ctx, sqlAddOrder, login, order, "NEW")

	return err
}
func (db *PGDataBase) GetOrder(ctx context.Context, login string) ([]models.ResponseOrder, error) {

	result := []models.ResponseOrder{}

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
			result = append(result, models.ResponseOrder{
				Order:      u.Order,
				Status:     u.Status,
				Accrual:    float32(intAccrual),
				UploadedAt: u.UploadedAt,
			})
		} else {
			result = append(result, models.ResponseOrder{
				Order:      u.Order,
				Status:     u.Status,
				UploadedAt: u.UploadedAt,
			})
		}

	}
	return result, nil
}
func (db *PGDataBase) GetBalance(ctx context.Context, login string) (models.BalanceResponse, error) {
	result := models.BalanceResponse{}
	sqlGetBalance := `SELECT current_value, withdrawed FROM wallet WHERE owner=$1;`

	query := db.conn.QueryRowContext(ctx, sqlGetBalance, login)

	//?????? ?????????? ???????? ???????????????? ??????????????????????
	err := query.Scan(&result.Current, &result.Withdrawn)

	if err != nil {
		log.Println("empty")
		return result, handlers.NewErrorWithDB(errors.New("empty"), "empty")
	}
	result = models.BalanceResponse{
		Current:   result.Current,
		Withdrawn: result.Withdrawn,
	}

	return result, nil
}
func (db *PGDataBase) Withdraw(ctx context.Context, login string, order string, value float32) error {
	db.UploadOrder(ctx, order, login)
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	sqlGetWallet := `SELECT current_value, withdrawed FROM wallet WHERE owner=$1;`
	result := GetWalletData{}
	query := tx.QueryRowContext(ctx, sqlGetWallet, login)
	err = query.Scan(&result.current, &result.withdrawed)
	if err != nil {
		log.Println(err)
		return err
	}
	f, err := strconv.ParseFloat(result.current, 32)
	if err != nil {
		log.Println(err)
		return err
	}
	if value > float32(f) {
		//402 ??? ???? ?????????? ???????????????????????? ??????????????;
		return handlers.NewErrorWithDB(errors.New("402"), "402")
	}
	//add this to withdraws.
	sqlAddWithdraws := `INSERT INTO withdraws (sum_withdrawed, order_temp, owner)
				  VALUES ($1, $2, $3)`

	withdraw := fmt.Sprintf("%f", value)
	_, err = tx.ExecContext(ctx, sqlAddWithdraws, withdraw, order, login)
	if err != nil {
		log.Println("err sqlAddWithdraws")
		return err
	}
	//increase wallet

	f2, err := strconv.ParseFloat(result.withdrawed, 32)
	if err != nil {
		log.Println("strconv.ParseFloat(result.withdrawed, 32)")
		return err
	}

	sqlUpdateWallet := `UPDATE wallet SET current_value = ($1), withdrawed = ($2) WHERE owner = ($3);`

	withdrawed := fmt.Sprintf("%f", float32(f2)+value)
	log.Println("withdrawed: ", withdrawed)

	current := float32(f) - value
	s := fmt.Sprintf("%f", current)
	_, err = tx.ExecContext(ctx, sqlUpdateWallet, s, withdrawed, login)

	if err != nil {
		log.Println("err sqlUpdateWallet")
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}
func (db *PGDataBase) GetOrderInfo(ctx context.Context, order string) (models.ResponseOrderInfo, error) {
	result := models.ResponseOrderInfo{}

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
	intAccrual, err := strconv.ParseFloat(u.Accrual, 32)
	if err != nil {
		log.Println(err)
		return result, err
	}
	result = models.ResponseOrderInfo{
		Order:   u.Order,
		Status:  u.Status,
		Accrual: float32(intAccrual),
	}

	return result, nil
}
func (db *PGDataBase) GetWithdraws(ctx context.Context, login string) ([]models.ResponseWithdraws, error) {
	result := []models.ResponseWithdraws{}

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
		result = append(result, models.ResponseWithdraws{
			Order:       u.Order,
			Sum:         float32(intSum),
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
	log.Println("GetNewOrder: Order: ", order)
	return order, nil
}
