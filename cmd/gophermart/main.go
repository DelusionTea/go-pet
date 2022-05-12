package main

import (
	"context"
	"database/sql"
	"github.com/DelusionTea/go-pet.git/cmd/conf"
	"github.com/DelusionTea/go-pet.git/internal/app/handlers"
	"github.com/DelusionTea/go-pet.git/internal/app/magic"
	"github.com/DelusionTea/go-pet.git/internal/app/middleware"
	"github.com/DelusionTea/go-pet.git/internal/database"
	"github.com/DelusionTea/go-pet.git/internal/workers"
	"github.com/gin-gonic/gin"
	"github.com/go-session/cookie"
	"github.com/go-session/session"
	"log"
	"net/http"
	"os"
	"os/signal"
)

var (
	hashKey = []byte("FF51A553-72FC-478B-9AEF-93D6F506DE91")
)

func setupMagic(repo handlers.MarketInterface, conf *conf.Config, wp *workers.Workers) *magic.Handler {
	magica := magic.New(repo, conf.ServerAddress, conf.SystemAccrualURL, wp)
	return magica
}
func setupRouter(repo handlers.MarketInterface, conf *conf.Config, wp *workers.Workers) *gin.Engine {
	//handler := handlers.New(handlers.MarketInterface, conf.ServerAddress, conf.ServerAddress, wp)
	session.InitManager(

		session.SetStore(
			cookie.NewCookieStore(
				cookie.SetCookieName("demo_cookie_store_id"),
				cookie.SetHashKey(hashKey),
			),
		),
	)

	router := gin.Default()
	router.Use(middleware.GzipEncodeMiddleware())
	router.Use(middleware.GzipDecodeMiddleware())
	handler := handlers.New(repo, conf.ServerAddress, conf.ServerAddress, wp)
	//go handler.AccrualAskWorker()
	router.POST("/api/user/register", handler.HandlerRegister)
	router.POST("/api/user/login", handler.HandlerLogin)
	router.POST("/api/user/orders", handler.HandlerPostOrders)
	router.GET("/api/user/orders", handler.HandlerGetOrders)
	router.GET("/api/user/balance", handler.HandlerGetBalance)
	router.POST("/api/user/balance/withdraw", handler.HandlerWithdraw)
	router.GET("/api/user/balance/withdrawals", handler.HandlerWithdraws)

	router.HandleMethodNotAllowed = true

	return router
}
func main() {

	ctx, cancel := context.WithCancel(context.Background())
	cfg := conf.GetConfig()
	var handler *gin.Engine
	wp := workers.New(ctx, cfg.NumbWorkers, cfg.WorkerBuff)
	go func() {
		wp.Run(ctx)
	}()
	db, err := sql.Open("postgres", cfg.DataBase)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	database.SetUpDataBase(db, ctx)
	log.Println(database.NewDatabaseRepository(db))
	handler = setupRouter(database.NewDatabase(db), cfg, wp)
	magica := setupMagic(database.NewDatabase(db), cfg, wp)
	server := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: handler,
	}
	go magica.AccrualAskWorker()
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
