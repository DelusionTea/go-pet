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
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

//const userkey = "user"

type DBError struct {
	Err   error
	Title string
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

type MarketInterface interface {
	UpdateStatus(status string, ctx context.Context)
	Register(login string, pass string, ctx context.Context) error
	Login(login string, pass string, ctx context.Context) (string, error)
	CheckAuth(login string, ctx context.Context) (string, error)
	UploadOrder(login string, order []byte, ctx context.Context) error
	GetOrder(login string, ctx context.Context) ([]ResponseOrder, error)
	GetBalance(status string, ctx context.Context)
	Withdraw(status string, ctx context.Context)
	GetWithdraws(status string, ctx context.Context)
}
type user struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	//IsLogined bool
}

type order struct {
	Owner      string    `json:"login"`
	Order      []byte    `json:"order"`
	Status     string    `json:"status"`
	Accrual    int       `json:"accrual"`
	UploadedAt time.Time `json:"uploaded_at"`
}
type ResponseOrder struct {
	Order      string    `json:"order"`
	Status     string    `json:"status"`
	Accrual    int       `json:"accrual"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type Handler struct {
	repo          MarketInterface
	serverAddress string
	wp            workers.Workers
}

func New(repo MarketInterface, serverAddress string, wp *workers.Workers) *Handler {
	return &Handler{
		repo:          repo,
		serverAddress: serverAddress,
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
		return
	}
	value := user{}
	defer c.Request.Body.Close()

	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, "Server Error")
		log.Println("Server Error  89")
		return
	}
	//if err := json.Unmarshal([]byte(body), &value); err != nil {
	//	c.IndentedJSON(http.StatusInternalServerError, "Server Error")
	//	log.Println("Server Error")
	//	return
	//}

	json.Unmarshal([]byte(body), &value)

	//response, err := h.repo.AddURLs(c.Request.Context(), data, c.GetString("userId"))

	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, "Server Error")
		log.Println("Server Error 104")
		return
	}

	//log.Println(value.Login, value.Password)
	//if err != nil {
	//	c.IndentedJSON(http.StatusInternalServerError, "Server Error")
	//	log.Println("Server Error")
	//	return
	//}
	if (value.Login == "") || (value.Password == "") {
		c.IndentedJSON(http.StatusBadRequest, "Error")
		log.Println("Bad Request Error 116")
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
	//TEMP
	//baseURL := "http://" + h.serverAddress
	//baseURL = baseURL + "/"
	//baseURL := "localhost"
	//c.SetCookie("user", value.Login, 864000, "/", baseURL, false, false)
	//
	//CheckCookie, err := c.Cookie("user") //c.Set("userId", id.String())
	//log.Println(CheckCookie, "- Проверка в функции Регистрации")
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
		return
	}
	//var results string
	value := user{}
	defer c.Request.Body.Close()

	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, "Server Error")
		log.Println("Server Error 165")
		return
	}

	json.Unmarshal([]byte(body), &value)
	//response, err := h.repo.AddURLs(c.Request.Context(), data, c.GetString("userId"))

	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, "Server Error")
		log.Println("Server Error 174")
		return
	}
	//if err := json.Unmarshal([]byte(body), &value); err != nil {
	//	c.IndentedJSON(http.StatusInternalServerError, "Server Error")
	//	log.Println("Server Error")
	//	return
	//}
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
		//if errors.As(err, &ue) && ue.Title == "user not found" {
		//	c.IndentedJSON(http.StatusConflict, "Status Conflict")
		//	return
		//}
		if errors.As(err, &ue) && (ue.Title == "wrong password" || ue.Title == "user not found") {
			c.IndentedJSON(http.StatusUnauthorized, "Status Unauthorized")
			log.Println("bad login pass")
			return
		}
		c.IndentedJSON(http.StatusInternalServerError, err)
		log.Println("Server Error 203")
		return
	}
	log.Println(results)
	//baseURL := "http://" + h.serverAddress
	//baseURL = baseURL + "/"
	//c.SetCookie("user", value.Login, 864000, "/", baseURL, false, false)
	//CheckCookie, err := c.Cookie("user") //c.Set("userId", id.String())
	//log.Println(CheckCookie, "- Проверка в функции Логина")
	store.Set("user", value.Login)
	if err := store.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		log.Println("Server Error 215")
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
		return
	}
	//result, err := h.repo.GetUserURL(c.Request.Context(), c.GetString("userId"))
	value := order{}
	//value.Owner, err = c.Cookie("user")
	//log.Println("value.Owner:  ", value.Owner)
	//if err != nil {
	//	log.Println("value.Owner:  ", value.Owner, "  we have error - empty")
	//	c.IndentedJSON(http.StatusUnauthorized, "Status Unauthorized")
	//	return
	//}
	value.Owner = fmt.Sprintf("%v", user)
	value.Order = body

	if !luhn.Valid(string(value.Order)) {
		c.IndentedJSON(http.StatusUnprocessableEntity, "Order is stupid! It's not real!! AHAHAHAHAHAAHAH")
		return
	}

	err = h.repo.UploadOrder(value.Owner, value.Order, c)
	//200 — номер заказа уже был загружен этим пользователем;
	//202 — новый номер заказа принят в обработку;
	//!!!400 — неверный формат запроса;
	//+++401 — пользователь не аутентифицирован;
	//409 — номер заказа уже был загружен другим пользователем;
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
			return
		}

	}
	c.IndentedJSON(http.StatusAccepted, "Accepted")
	log.Println("Accepted")

	//CHECK BODY VALUE!!!!

	//user := user{}
	//h.repo.CheckAuth(user.Login, c)
	//Номер заказа может быть проверен на корректность ввода с помощью алгоритма Луна.
	//	Формат запроса:
	//POST /api/user/orders HTTP/1.1
	//Content-Type: text/plain
	//...
	//
	//12345678903
	//Возможные коды ответа:
	//200 — номер заказа уже был загружен этим пользователем;
	//202 — новый номер заказа принят в обработку;
	//!!!400 — неверный формат запроса;
	//+++401 — пользователь не аутентифицирован;
	//409 — номер заказа уже был загружен другим пользователем;
	//!!!!!!!!!!!!!!422 — неверный формат номера заказа;!!!!!!!!!!!!!!!!!
	//500 — внутренняя ошибка сервера.
}

func (h *Handler) HandlerGetOrders(c *gin.Context) {
	store, err := session.Start(context.Background(), c.Writer, c.Request)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, "Server Error")
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
		return
	}
	if len(result) == 0 {
		c.IndentedJSON(http.StatusNoContent, result)
		return
	}
	c.IndentedJSON(http.StatusOK, result)
	//	Хендлер доступен только авторизованному пользователю. Номера заказа в выдаче должны быть отсортированы по времени загрузки от самых старых к самым новым. Формат даты — RFC3339.
	//		Доступные статусы обработки расчётов:
	//	NEW — заказ загружен в систему, но не попал в обработку;
	//	PROCESSING — вознаграждение за заказ рассчитывается;
	//	INVALID — система расчёта вознаграждений отказала в расчёте;
	//	PROCESSED — данные по заказу проверены и информация о расчёте успешно получена.
	//		Формат запроса:
	//	GET /api/user/orders HTTP/1.1
	//	Content-Length: 0
	//	Возможные коды ответа:
	//	200 — успешная обработка запроса.
	//		Формат ответа:
	//	200 OK HTTP/1.1
	//	Content-Type: application/json
	//	...
	//
	//[
	//	{
	//	"number": "9278923470",
	//	"status": "PROCESSED",
	//	"accrual": 500,
	//	"uploaded_at": "2020-12-10T15:15:45+03:00"
	//	},
	//	{
	//	"number": "12345678903",
	//	"status": "PROCESSING",
	//	"uploaded_at": "2020-12-10T15:12:01+03:00"
	//	},
	//	{
	//	"number": "346436439",
	//	"status": "INVALID",
	//	"uploaded_at": "2020-12-09T16:09:53+03:00"
	//	}
	//	]
	//
	//	204 — нет данных для ответа.
	//	401 — пользователь не авторизован.
	//	500 — внутренняя ошибка сервера
}
func (h *Handler) HandlerGetBalance(c *gin.Context) {
	store, err := session.Start(context.Background(), c.Writer, c.Request)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, "Server Error")
		log.Println("Server Error 227")
		return
	}
	user, ok := store.Get("user")
	log.Println("user is......", fmt.Sprintf("%v", user))
	if user == nil || (!ok) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	//Хендлер доступен только авторизованному пользователю. В ответе должны содержаться данные о текущей сумме баллов лояльности, а также сумме использованных за весь период регистрации баллов.
	//	Формат запроса:
	//GET /api/user/balance HTTP/1.1
	//Content-Length: 0
	//Возможные коды ответа:
	//200 — успешная обработка запроса.
	//	Формат ответа:
	//200 OK HTTP/1.1
	//Content-Type: application/json
	//...
	//
	//{
	//	"current": 500.5,
	//	"withdrawn": 42
	//}
	//
	//401 — пользователь не авторизован.
	//500 — внутренняя ошибка сервера.
}
func (h *Handler) HandlerWithdraw(c *gin.Context) {
	store, err := session.Start(context.Background(), c.Writer, c.Request)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, "Server Error")
		log.Println("Server Error 227")
		return
	}
	user, ok := store.Get("user")
	log.Println("user is......", fmt.Sprintf("%v", user))
	if user == nil || (!ok) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	//Хендлер доступен только авторизованному пользователю. Номер заказа представляет собой гипотетический номер нового заказа пользователя, в счёт оплаты которого списываются баллы.
	//	Примечание: для успешного списания достаточно успешной регистрации запроса, никаких внешних систем начисления не предусмотрено и не требуется реализовывать.
	//	Формат запроса:
	//POST /api/user/balance/withdraw HTTP/1.1
	//Content-Type: application/json
	//
	//{
	//	"order": "2377225624",
	//	"sum": 751
	//}
	//Здесь order — номер заказа, а sum — сумма баллов к списанию в счёт оплаты.
	//	Возможные коды ответа:
	//200 — успешная обработка запроса;
	//401 — пользователь не авторизован;
	//402 — на счету недостаточно средств;
	//422 — неверный номер заказа;
	//500 — внутренняя ошибка сервера.
}
func (h *Handler) HandlerWithdraws(c *gin.Context) {
	store, err := session.Start(context.Background(), c.Writer, c.Request)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, "Server Error")
		log.Println("Server Error 227")
		return
	}
	user, ok := store.Get("user")
	log.Println("user is......", fmt.Sprintf("%v", user))
	if user == nil || (!ok) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	//	Хендлер доступен только авторизованному пользователю. Факты выводов в выдаче должны быть отсортированы по времени вывода от самых старых к самым новым. Формат даты — RFC3339.
	//		Формат запроса:
	//	GET /api/user/withdrawals HTTP/1.1
	//	Content-Length: 0
	//	Возможные коды ответа:
	//	200 — успешная обработка запроса.
	//		Формат ответа:
	//	200 OK HTTP/1.1
	//	Content-Type: application/json
	//	...
	//
	//[
	//	{
	//	"order": "2377225624",
	//	"sum": 500,
	//	"processed_at": "2020-12-09T16:09:57+03:00"
	//	}
	//	]
	//
	//	204 — нет ни одного списания.
	//	401 — пользователь не авторизован.
	//	500 — внутренняя ошибка сервера.
}
func (h *Handler) HandlerGetInfo(c *gin.Context) {
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
