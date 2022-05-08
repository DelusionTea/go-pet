package main

import (
	"context"
	"database/sql"
	"github.com/DelusionTea/go-pet.git/cmd/conf"
	"github.com/DelusionTea/go-pet.git/internal/app/handlers"
	"github.com/DelusionTea/go-pet.git/internal/app/middleware"
	"github.com/DelusionTea/go-pet.git/internal/database"
	"github.com/DelusionTea/go-pet.git/internal/workers"
	"github.com/go-session/redis/v3"
	"github.com/go-session/session/v3"

	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"os/signal"
)

var secret = []byte("secret")

func setupRouter(repo handlers.MarketInterface, conf *conf.Config, wp *workers.Workers) *gin.Engine {
	session.InitManager(
		session.SetStore(redis.NewRedisStore(&redis.Options{
			Addr: "localhost:6379",
			DB:   15, //Что это за строка-то?
		})),
	)
	/*func setupRouter(repo memory.MemoryMap, baseURL string, conf *conf.Config) *gin.Engine {*/
	router := gin.Default()
	//router.Use(sessions.Sessions("mysession", sessions.NewCookieStore(secret)))
	//store := cookie.NewStore([]byte("secret"))
	//store.Options(sessions.Options{MaxAge:   60 * 60 * 24}) // expire in a day
	//router.Use(sessions.Sessions("mysession", store))
	router.Use(middleware.GzipEncodeMiddleware())
	router.Use(middleware.GzipDecodeMiddleware())
	//router.Use(middleware.CookieMiddleware(conf))
	//router.Use(gzip.Gzip(gzip.DefaultCompression))
	handler := handlers.New(repo, conf.ServerAddress, wp)
	router.POST("/api/user/register", handler.HandlerRegister)
	router.POST("/api/user/login", handler.HandlerLogin)
	router.POST("/api/user/orders", handler.HandlerPostOrders)
	router.GET("/api/user/orders", handler.HandlerGetOrders)
	router.GET("/api/user/balance", handler.HandlerGetBalance)
	router.POST("/api/user/balance/withdraw", handler.HandlerWithdraw)
	router.GET("/api/user/balance/withdrawals", handler.HandlerWithdraws)
	router.GET("/api/orders/:number", handler.HandlerGetInfo)
	//router.GET("/:id", handler.HandlerGetURLByID)
	//router.POST("/", handler.HandlerCreateShortURL)
	//router.POST("/api/shorten", handler.HandlerShortenURL)
	//router.GET("/ping", handler.HandlerPingDB)
	//router.GET("/api/user/urls", handler.HandlerHistoryOfURLs)
	////POST /api/shorten/batch
	//router.POST("/api/shorten/batch", handler.HandlerBatch)
	//router.DELETE("/api/user/urls", handler.DeleteBatch)

	router.HandleMethodNotAllowed = true

	return router
}
func main() {

	ctx, cancel := context.WithCancel(context.Background())
	cfg := conf.GetConfig()
	var handler *gin.Engine
	//db, err := sql.Open("postgres", cfg.DataBase)
	wp := workers.New(ctx, cfg.NumbWorkers, cfg.WorkerBuff)
	go func() {
		wp.Run(ctx)
	}()
	//if cfg.DataBase != "" {

	db, err := sql.Open("postgres", cfg.DataBase)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	database.SetUpDataBase(db, ctx)
	log.Println(database.NewDatabaseRepository(db))
	handler = setupRouter(database.NewDatabase(db), cfg, wp)
	//	//handler = setupRouter(memory.NewMemoryFile(cfg.FilePath, cfg.BaseURL), cfg.BaseURL, cfg)
	//} else {
	//	handler = setupRouter(memory.NewMemoryFile(ctx, cfg.FilePath, cfg.BaseURL), cfg, wp)
	//}
	server := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: handler,
	}

	go func() {
		log.Fatal(server.ListenAndServe())
		cancel()
	}()

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)
	select {
	case <-sigint:
		cancel()
	case <-ctx.Done():
	}
	server.Shutdown(context.Background())
}
