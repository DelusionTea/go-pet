package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/DelusionTea/go-pet.git/internal/database/models"
	"github.com/DelusionTea/go-pet.git/internal/luhn"
	"github.com/DelusionTea/go-pet.git/internal/workers"
	"github.com/gin-gonic/gin"
	"github.com/go-session/session/v3"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type MarketInterface interface {
	UpdateStatus(ctx context.Context, order string, status string) error
	Register(ctx context.Context, login string, pass string) error
	Login(ctx context.Context, login string, pass string) (string, error)
	UploadOrder(ctx context.Context, login string, order string) error
	GetOrder(ctx context.Context, login string) ([]models.ResponseOrder, error)
	GetBalance(ctx context.Context, login string) (models.BalanceResponse, error)
	Withdraw(ctx context.Context, login string, order string, value float32) error
	GetWithdraws(ctx context.Context, login string) ([]models.ResponseWithdraws, error)
	UpdateWallet(ctx context.Context, order string, value float32) error
	GetOrderInfo(ctx context.Context, order string) (models.ResponseOrderInfo, error)
	UpdateAccural(ctx context.Context, order string, accural float32) error
	GetNewOrder(ctx context.Context) (string, error)
}
type user struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	//IsLogined bool
}

type order struct {
	Owner      string    `json:"login"`
	Order      string    `json:"order"`
	Status     string    `json:"status"`
	Accrual    float32   `json:"accrual"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type Handler struct {
	repo          MarketInterface
	serverAddress string
	accuralURL    string
	wp            workers.Workers
}
type DBError struct {
	Err   error
	Title string
}

type RequestWithdraw struct {
	Order string  `json:"order"`
	Sum   float32 `json:"sum"`
}

func (err *DBError) Error() string {
	return fmt.Sprintf("%v", err.Err)
}

func NewErrorWithDB(err error, title string) error {
	return &DBError{
		Err:   err,
		Title: title,
	}
}

func New(repo MarketInterface, serverAddress string, accrualURL string, wp *workers.Workers) *Handler {
	return &Handler{
		repo:          repo,
		serverAddress: serverAddress,
		accuralURL:    accrualURL,
		wp:            *wp,
	}
}

func (h *Handler) HandlerRegister(c *gin.Context) {
	log.Println("Register Start")
	//session := sessions.Default(c)
	store, err := session.Start(context.Background(), c.Writer, c.Request)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, "Server Error")
		log.Println("Server Error 80")
		log.Println(err)
		return
	}
	value := user{}
	defer c.Request.Body.Close()

	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, "Server Error")
		log.Println("Server Error  89")
		log.Println(err)
		return
	}

	err = json.Unmarshal(body, &value)

	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, "Server Error")
		log.Println("Server Error 104")
		return
	}

	if (value.Login == "") || (value.Password == "") {
		c.IndentedJSON(http.StatusBadRequest, "Error")
		log.Println("Bad Request Error 116")
		log.Println(err)
		return
	}
	err = h.repo.Register(c, value.Login, value.Password)
	if err != nil {
		var ue *DBError
		if errors.As(err, &ue) && ue.Title == "Conflict" {
			c.IndentedJSON(http.StatusConflict, "Status Conflict")
			log.Println("Conflict  124")
			return
		}
		c.IndentedJSON(http.StatusInternalServerError, err)
		log.Println("Server Error 128")
		return
	}
	c.IndentedJSON(http.StatusOK, "Success Register")
	log.Println("OK call Login")

	store.Set("user", value.Login)
	if err := store.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		log.Println("Server Error 144")
		log.Println(err.Error())
		return
	}

}
func (h *Handler) HandlerLogin(c *gin.Context) {
	log.Println("Login Start")
	store, err := session.Start(context.Background(), c.Writer, c.Request)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, "Server Error")
		log.Println("Server Error 155")
		log.Println(err)
		return
	}
	//var results string
	value := user{}
	defer c.Request.Body.Close()

	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, "Server Error")
		log.Println("Server Error 165")
		log.Println(err)
		return
	}

	err = json.Unmarshal(body, &value)

	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, "Server Error")
		log.Println("Server Error 174")
		log.Println(err)
		return
	}

	log.Println(value.Login, value.Password)

	if (value.Login == "") || (value.Password == "") {
		c.IndentedJSON(http.StatusBadRequest, "Error")
		log.Println("Bad Request Error 186  Login:", value.Login, "  Passwprd:  ", value.Password)
		return
	}
	results, err := h.repo.Login(c, value.Login, value.Password)
	//
	if err != nil {
		var ue *DBError

		if errors.As(err, &ue) && (ue.Title == "wrong password" || ue.Title == "user not found") {
			c.IndentedJSON(http.StatusUnauthorized, "Status Unauthorized")
			log.Println("bad login pass")
			return
		}
		c.IndentedJSON(http.StatusInternalServerError, err)
		log.Println("Server Error 203")
		log.Println(err)
		return
	}
	log.Println(results)

	store.Set("user", value.Login)
	if err := store.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		log.Println("Server Error 215")
		log.Println(err)
		return
	}
	c.IndentedJSON(http.StatusOK, "Success Login")
	log.Println("ok")

}

func (h *Handler) HandlerPostOrders(c *gin.Context) {
	log.Println("Start HandlerPostOrders")
	store, err := session.Start(context.Background(), c.Writer, c.Request)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, "Server Error")
		log.Println("Server Error 227")
		log.Println(err)
		return
	}
	user, ok := store.Get("user")
	log.Println("user is......", fmt.Sprintf("%v", user))
	if user == nil || (!ok) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	defer c.Request.Body.Close()

	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, "Server Error")
		log.Println("Server Error 241")
		log.Println(err)
		return
	}

	value := order{}
	value.Owner = fmt.Sprintf("%v", user)
	value.Order = string(body)

	if !luhn.IsValid(string(value.Order)) {
		c.IndentedJSON(http.StatusUnprocessableEntity, "Order is stupid! It's not real!! AHAHAHAHAHAAHAH")
		return
	}

	err = h.repo.UploadOrder(c, value.Owner, value.Order)

	if err != nil {
		var ue *DBError
		if errors.As(err, &ue) && (ue.Title == "Already here") {
			c.IndentedJSON(http.StatusOK, "Already here")
			log.Println("Already here")
			return
		}
		if errors.As(err, &ue) && (ue.Title == "Conflict") {
			c.IndentedJSON(http.StatusConflict, "Conflict")
			log.Println("Conflict")
			return
		} else {
			c.IndentedJSON(http.StatusInternalServerError, err)
			log.Println("Server Error 275")
			log.Println(err)
			return
		}

	}
	c.IndentedJSON(http.StatusAccepted, "Accepted")
	//h.CalculateThings(value.Order, c)
	//h.AccrualAskWorkerRunner(c)
	log.Println("Accepted")
	//go h.AccrualAskWorker(c)
}

func (h *Handler) HandlerGetOrders(c *gin.Context) {
	log.Println("start HandlerGetOrders")
	store, err := session.Start(context.Background(), c.Writer, c.Request)

	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, "Server Error")
		log.Println(err)
		//log.Println("Server Error 227")
		return
	}

	user, ok := store.Get("user")
	log.Println("user is......", fmt.Sprintf("%v", user))
	if user == nil || (!ok) {

		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return

	}

	result, err := h.repo.GetOrder(c, fmt.Sprintf("%v", user))
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, err)
		log.Println(err)
		return
	}
	if len(result) == 0 {
		c.IndentedJSON(http.StatusNoContent, result)
		return
	}
	log.Println("Result of HandlerGetOrders :", result)

	c.JSON(http.StatusOK, result)
}
func (h *Handler) HandlerGetBalance(c *gin.Context) {
	log.Println("Start HandlerGetBalance")
	store, err := session.Start(context.Background(), c.Writer, c.Request)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, "Server Error")
		log.Println("Server Error 227")
		log.Println(err)
		return
	}
	user, ok := store.Get("user")
	log.Println("user is......", fmt.Sprintf("%v", user))
	if user == nil || (!ok) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	result, err := h.repo.GetBalance(c, fmt.Sprintf("%v", user))
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, err)
		log.Println(err)
		return
	}

	log.Println("HandlerGetBalance result:", result)

	c.JSON(http.StatusOK, result)

}

func (h *Handler) HandlerWithdraw(c *gin.Context) {
	log.Println("Start HandlerWithdraw")
	store, err := session.Start(context.Background(), c.Writer, c.Request)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, "Server Error")
		log.Println("Server Error 227")
		log.Println(err)
		return
	}
	user, ok := store.Get("user")
	log.Println("user is......", fmt.Sprintf("%v", user))
	if user == nil || (!ok) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	value := RequestWithdraw{}
	defer c.Request.Body.Close()

	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, "Server Error")
		log.Println("Server Error  89")
		log.Println(err)
		return
	}
	log.Println(body)

	err = json.Unmarshal(body, &value)

	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, "Server Error")
		log.Println("Server Error 434", err)
		log.Println(err)
		return
	}

	if !luhn.IsValid(string(value.Order)) {
		c.IndentedJSON(http.StatusUnprocessableEntity, "Order is stupid! It's not real!! AHAHAHAHAHAAHAH")
		return
	}
	log.Println("call db Withdraw")
	err = h.repo.Withdraw(c, fmt.Sprintf("%v", user), string(value.Order), value.Sum)

	if err != nil {
		var ue *DBError
		if errors.As(err, &ue) && (ue.Title == "402") {
			c.IndentedJSON(http.StatusPaymentRequired, "PaymentRequired")
			log.Println("PaymentRequired")
			return
		} else {
			c.IndentedJSON(http.StatusInternalServerError, err)
			log.Println("Server Error 452", err)
			return
		}

	}
	log.Println("End of HandlerWithdraw")
	c.IndentedJSON(http.StatusOK, "Ok")

}
func (h *Handler) HandlerWithdraws(c *gin.Context) {
	log.Println("Start of Handler Withdraws")
	store, err := session.Start(context.Background(), c.Writer, c.Request)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, "Server Error")
		log.Println("Server Error 227")
		log.Println(err)
		return
	}
	user, ok := store.Get("user")
	log.Println("user is......", fmt.Sprintf("%v", user))
	if user == nil || (!ok) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	result, err := h.repo.GetWithdraws(c, fmt.Sprintf("%v", user))
	//result, err := h.repo.GetOrder(fmt.Sprintf("%v", user), c.Request.Context())
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, err)
		log.Println(err)
		return
	}
	if len(result) == 0 {
		c.IndentedJSON(http.StatusNoContent, result)
		log.Println("len(res) is nil")
		return
	}
	log.Println("WITHDRAWS:")
	log.Println("Result of Handler Withdraws:")
	log.Println(result)

	c.JSON(http.StatusOK, result)

}
