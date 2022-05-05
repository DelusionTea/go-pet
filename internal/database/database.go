package database

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"log"
)

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
	sqlCreateDB := `CREATE TABLE IF NOT EXISTS urls (
								id serial PRIMARY KEY,
								user_id uuid DEFAULT uuid_generate_v4 (), 	
								origin_url VARCHAR NOT NULL, 
								short_url VARCHAR NOT NULL UNIQUE,
								is_deleted BOOLEAN NOT NULL DEFAULT FALSE
					);`
	res, err := db.ExecContext(ctx, sqlCreateDB)
	log.Println("Create table", err, res)
	return nil
}

type PGDataBase struct {
	conn       *sql.DB
	baseURL    string
	DeleteFlag bool `json:"delete_flag"`
}

func NewDatabase(baseURL string, db *sql.DB) *PGDataBase {
	result := &PGDataBase{
		conn:    db,
		baseURL: baseURL,
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

type GetURLdata struct {
	OriginURL string
	IsDeleted bool
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

func (db *PGDataBase) DeleteManyURL(ctx context.Context, urls []string, user string) error {

	sql := `UPDATE urls SET is_deleted = true WHERE short_url = ANY ($1);`
	urlsToDelete := []string{}
	for _, url := range urls {
		if db.isOwner(ctx, url, user) {
			urlsToDelete = append(urlsToDelete, url)
		}
	}
	_, err := db.conn.ExecContext(ctx, sql, pq.Array(urlsToDelete))
	if err != nil {
		return err
	}
	return nil
}

func (db *PGDataBase) isOwner(ctx context.Context, url string, user string) bool {
	sqlGetURLRow := `SELECT user_id FROM urls WHERE short_url=$1 FETCH FIRST ROW ONLY;`
	query := db.conn.QueryRowContext(ctx, sqlGetURLRow, url)
	result := ""
	query.Scan(&result)
	return result == user
}

func (db *PGDataBase) DeleteURLs(ctx context.Context, urls []string, user string) error {
	sql := `UPDATE urls SET is_deleted = true WHERE short_url = ANY ($1);`
	urlsToDelete := []string{}
	for _, url := range urls {
		if db.isOwner(ctx, url, user) {
			urlsToDelete = append(urlsToDelete, url)
		}
	}
	_, err := db.conn.ExecContext(ctx, sql, pq.Array(urlsToDelete))
	if err != nil {
		return err
	}
	return nil
}
