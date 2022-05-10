package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/DelusionTea/go-pet.git/internal/luhn"
	"github.com/DelusionTea/go-pet.git/internal/workers"
	"github.com/gin-gonic/gin"
	"github.com/go-session/session/v3"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

//const userkey = "user"
type MarketInterface interface {
	UpdateStatus(order string, status string, ctx context.Context) error
	Register(login string, pass string, ctx context.Context) error
	Login(login string, pass string, ctx context.Context) (string, error)
	CheckAuth(login string, ctx context.Context) (string, error)
	UploadOrder(login string, order string, ctx context.Context) error
	GetOrder(login string, ctx context.Context) ([]ResponseOrder, error)
	GetBalance(login string, ctx context.Context) (BalanceResponse, error)
	Withdraw(login string, order string, value float64, ctx context.Context) error
	GetWithdraws(login string, ctx context.Context) ([]ResponseWithdraws, error)
	UpdateWallet(order string, value float64, ctx context.Context) error
	GetOrderInfo(order string, ctx context.Context) (ResponseOrderInfo, error)
	UpdateAccural(order string, accural string, ctx context.Context) error
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
	Accrual    int       `json:"accrual"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type ResponseOrder struct {
	Order      string    `json:"number"`
	Status     string    `json:"status"`
	Accrual    int       `json:"accrual"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type ResponseOrderInfo struct {
	Order   string `json:"number"`
	Status  string `json:"status"`
	Accrual int    `json:"accrual"`
}

type ResponseWithdraws struct {
	Order       string    `json:"order"`
	Sum         int       `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}
type BalanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn int     `json:"withdrawn"`
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
	Sum   float64 `json:"sum"`
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

type ResponseAccural struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float32 `json:"accrual"`
}

//func (h *Handler) AccrualAskWorker(co *gin.Context) {
//	c := time.Tick(time.Second)
//	for range c {
//		go h.AccrualAskWorkerRunner(co)
//	}
//}
func (h *Handler) AccrualAskWorkerRunner(c *gin.Context) {
	log.Println("START FUCKIG ROUTINE")
	order, err := h.repo.GetNewOrder(c)
	if err != nil {
		//c.IndentedJSON(http.StatusInternalServerError, "Server Error")
		log.Println("Server Error  125")
		log.Println(err)
		return
	}
	log.Println("ORDER OF FUCKIG ROUTINE is ", order)
	if order != "" {
		log.Println("start celculate things order: ", order)
		//Принять заказ и изменить статус на "в обработке"
		value := ResponseAccural{}
		url := "http://" + h.accuralURL + "/api/orders/" + order
		log.Println("URL:")
		log.Println(url)
		if (value.Status != "INVALID") || (value.Status != "PROCESSED") {
			log.Println("(value.Status != \"INVALID\") || (value.Status != \"PROCESSED\")")
			response, err := http.Get(url) //
			defer response.Body.Close()
			body, err := ioutil.ReadAll(response.Body)
			if err != nil {
				//c.IndentedJSON(http.StatusInternalServerError, "Server Error")
				log.Println("Server Error  89")
				log.Println(err)
				return
			}

			err = json.Unmarshal(body, &value)
			log.Println("body: ", body)
			log.Println("status is:", &value.Status)
			log.Println("accrual is:", &value.Accrual)
			log.Println("order is:", &value.Order)
			if value.Status == "PROCESSING" {
				h.repo.UpdateStatus(order, "PROCESSING", c)
				log.Println("UpdateStatus(order, \"PROCESSING\"")
			}
		}

		if value.Status == "INVALID" {
			log.Println("value.Status == \"INVALID\"")
			h.repo.UpdateStatus(order, "INVALID", c)
			log.Println("UpdateStatus(order, \"INVALID\"")
			return
		}
		//call this thing

		h.repo.UpdateStatus(order, "PROCESSING", c)
		log.Println("UpdateStatus(order, \"PROCESSING\", c)", order)
		log.Println("start Magic")

		//Начислить баллы
		log.Println("Start Update Wallet")
		err := h.repo.UpdateWallet(order, float64(value.Accrual), c)
		if err != nil {
			h.repo.UpdateStatus(order, "INVALID", c)
			log.Println("UpdateStatus(order, \"INVALID\"")
			log.Println(err)
			return
		}
		//Изменить статус
		s := fmt.Sprintf("%f", float64(value.Accrual))
		err = h.repo.UpdateAccural(order, s, c)
		log.Println("UpdateAccural")
		if err != nil {
			h.repo.UpdateStatus(order, "INVALID", c)
			log.Println("UpdateStatus(order, \"INVALID\"")
			log.Println(err)
			return
		}
		err = h.repo.UpdateStatus(order, "PROCESSED", c)
		log.Println("UpdateStatus(order, \"PROCESSED\"")

	}
}
func (h *Handler) CalculateThings(order string, c *gin.Context) {
	log.Println("start celculate things order: ", order)
	//Принять заказ и изменить статус на "в обработке"
	value := ResponseAccural{}
	url := "http://" + h.accuralURL + "/api/orders/" + order
	log.Println("URL:")
	log.Println(url)
	for (value.Status != "INVALID") || (value.Status != "PROCESSED") {
		response, err := http.Get(url) //
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError, "Server Error")
			log.Println("Server Error  89")
			log.Println(err)
			return
		}

		err = json.Unmarshal(body, &value)
		log.Println("body: ", body)
		log.Println("status is:", &value.Status)
		log.Println("accrual is:", &value.Accrual)
		log.Println("order is:", &value.Order)
		if value.Status == "PROCESSING" {
			h.repo.UpdateStatus(order, "PROCESSING", c)
			log.Println("UpdateStatus(order, \"INVALID\"")
			time.Sleep(1 * time.Second)
		}
	}

	if value.Status == "INVALID" {
		h.repo.UpdateStatus(order, "INVALID", c)
		log.Println("UpdateStatus(order, \"INVALID\"")
		return
	}
	//call this thing

	h.repo.UpdateStatus(order, "PROCESSING", c)
	log.Println("UpdateStatus(order, \"PROCESSING\", c)", order)
	log.Println("start Magic")

	//Начислить баллы
	log.Println("Start Update Wallet")
	err := h.repo.UpdateWallet(order, float64(value.Accrual), c)
	if err != nil {
		h.repo.UpdateStatus(order, "INVALID", c)
		log.Println("UpdateStatus(order, \"INVALID\"")
		log.Println(err)
		return
	}
	//Изменить статус
	s := fmt.Sprintf("%f", float64(value.Accrual))
	err = h.repo.UpdateAccural(order, s, c)
	log.Println("UpdateAccural")
	if err != nil {
		h.repo.UpdateStatus(order, "INVALID", c)
		log.Println("UpdateStatus(order, \"INVALID\"")
		log.Println(err)
		return
	}
	err = h.repo.UpdateStatus(order, "PROCESSED", c)
	log.Println("UpdateStatus(order, \"PROCESSED\"")

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
	err = h.repo.Register(value.Login, value.Password, c)
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

	return
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

	json.Unmarshal(body, &value)

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
	results, err := h.repo.Login(value.Login, value.Password, c)
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

	if !luhn.Valid(string(value.Order)) {
		c.IndentedJSON(http.StatusUnprocessableEntity, "Order is stupid! It's not real!! AHAHAHAHAHAAHAH")
		return
	}

	err = h.repo.UploadOrder(value.Owner, value.Order, c)

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
	go h.AccrualAskWorkerRunner(c)
	//h.AccrualAskWorkerRunner(c)
	log.Println("Accepted")
	//go h.AccrualAskWorker(c)
}

func (h *Handler) HandlerGetOrders(c *gin.Context) {
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

	result, err := h.repo.GetOrder(fmt.Sprintf("%v", user), c.Request.Context())
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, err)
		log.Println(err)
		return
	}
	if len(result) == 0 {
		c.IndentedJSON(http.StatusNoContent, result)
		return
	}
	log.Println(result)

	c.JSON(http.StatusOK, result)
}
func (h *Handler) HandlerGetBalance(c *gin.Context) {
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
	result, err := h.repo.GetBalance(fmt.Sprintf("%v", user), c.Request.Context())
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, err)
		log.Println(err)
		return
	}

	log.Println(result)

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

	if !luhn.Valid(string(value.Order)) {
		c.IndentedJSON(http.StatusUnprocessableEntity, "Order is stupid! It's not real!! AHAHAHAHAHAAHAH")
		return
	}
	log.Println("call db Withdraw")
	err = h.repo.Withdraw(fmt.Sprintf("%v", user), string(value.Order), value.Sum, c)

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
	log.Println("End of Handler Withdraws")
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

	result, err := h.repo.GetWithdraws(fmt.Sprintf("%v", user), c.Request.Context())
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
	log.Println(result)

	c.JSON(http.StatusOK, result)

}
func (h *Handler) HandlerGetInfo(c *gin.Context) {
	locOrder := c.Param("number")
	result, err := h.repo.GetWithdraws(locOrder, c)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, err)
		log.Println(err)
		return
	}
	c.JSON(http.StatusOK, result)
	//Формат запроса:
	//GET /api/orders/{number} HTTP/1.1
	//Content-Length: 0
	//Возможные коды ответа:
	//200 — успешная обработка запроса.
	//	Формат ответа:
	//200 OK HTTP/1.1
	//Content-Type: application/json
	//...
	//
	//{
	//	"order": "<number>",
	//	"status": "PROCESSED",
	//	"accrual": 500
	//}
	//
	//Поля объекта ответа:
	//order — номер заказа;
	//status — статус расчёта начисления:
	//REGISTERED — заказ зарегистрирован, но не начисление не рассчитано;
	//INVALID — заказ не принят к расчёту, и вознаграждение не будет начислено;
	//PROCESSING — расчёт начисления в процессе;
	//PROCESSED — расчёт начисления окончен;
	//accrual — рассчитанные баллы к начислению, при отсутствии начисления — поле отсутствует в ответе.
	//429 — превышено количество запросов к сервису.
	//	Формат ответа:
	//429 Too Many Requests HTTP/1.1
	//Content-Type: text/plain
	//Retry-After: 60
	//
	//No more than N requests per minute allowed
	//
	//500 — внутренняя ошибка сервера.
	//	Заказ может быть взят в расчёт в любой момент после его совершения. Время выполнения расчёта системой не регламентировано. Статусы INVALID и PROCESSED являются окончательными.
	//	Общее количество запросов информации о начислении не ограничено.
}
