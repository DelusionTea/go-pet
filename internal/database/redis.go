package database

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
)

type Redis struct {
	Client *redis.Client
}

var (
	ErrNil = errors.New("no matching record found in redis database")
	Ctx    = context.TODO()
)

func NewRedisDatabase(address string) (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: "",
		DB:       0,
	})
	if err := client.Ping(Ctx).Err(); err != nil {
		return nil, err
	}
	return &Redis{
		Client: client,
	}, nil
}

//type User struct {
//	user string `json:"user" binding:"required"`
//}
//
//func (db *Redis) SaveUser(user *User) error {
//	member := redis.Z{
//		Score:  float64(user.Points),
//		Member: user.Username,
//	}
//	pipe := db.Client.TxPipeline()
//	pipe.ZAdd("leaderboard", member)
//	rank := pipe.ZRank(leaderboardKey, user.Username)
//	_, err := pipe.Exec()
//	if err != nil {
//		return err
//	}
//	user.Rank = int(rank.Val())
//	return nil
//}
//
//func (db *Redis) GetUser(username string) (*User, error) {
//	pipe := db.Client.TxPipeline()
//	score := pipe.ZScore(leaderboardKey, username)
//	rank := pipe.ZRank(leaderboardKey, username)
//	_, err := pipe.Exec()
//	if err != nil {
//		return nil, err
//	}
//	if score == nil {
//		return nil, ErrNil
//	}
//	return &User{
//		Username: username,
//		Points:   int(score.Val()),
//		Rank:     int(rank.Val()),
//	}, nil
//}
