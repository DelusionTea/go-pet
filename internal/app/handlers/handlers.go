package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/DelusionTea/go-pet.git/internal/workers"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"log"
	"net/http"
)

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
	UploadOrder(status string, ctx context.Context)
	GetOrder(status string, ctx context.Context)
	GetBalance(status string, ctx context.Context)
	Withdraw(status string, ctx context.Context)
	GetWithdraws(status string, ctx context.Context)
}
type user struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type Handler struct {
	repo MarketInterface
	wp   workers.Workers
}

func New(repo MarketInterface, wp *workers.Workers) *Handler {
	return &Handler{
		repo: repo,
		wp:   *wp,
	}
}

func (h *Handler) HandlerRegister(c *gin.Context) {
	value := user{}
	defer c.Request.Body.Close()

	body, err := ioutil.ReadAll(c.Request.Body)

	if err := json.Unmarshal([]byte(body), &value); err != nil {
		c.IndentedJSON(http.StatusInternalServerError, "Server Error")
	}

	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, "Server Error")
		return
	}
	if (value.Login == "") || (value.Password == "") {
		c.IndentedJSON(http.StatusBadRequest, "Error")
		return
	}
	err = h.repo.Register(value.Login, value.Password, c)
	if err != nil {
		var ue *DBError
		if errors.As(err, &ue) && ue.Title == "Conflict" {
			c.IndentedJSON(http.StatusConflict, "Status Conflict")
			return
		}
		c.IndentedJSON(http.StatusInternalServerError, err)
		return
	}
	//Добавить в базу
	//Если имя занято вернуть ошибку
	//При проблемах добавления - тоже бросить ошибку
	//
	//shortURL := shorter.Shorter(url.URL)
	//err = h.repo.AddURL(c.Request.Context(), url.URL, shortURL, c.GetString("userId"))
	//if err != nil {
	//	var ue *DBError
	//	if errors.As(err, &ue) && ue.Title == "UniqConstraint" {
	//		result["result"] = h.baseURL + shortURL
	//		c.IndentedJSON(http.StatusConflict, result)
	//		return
	//	}
	//	c.IndentedJSON(http.StatusInternalServerError, err)
	//	return
	//}
	//result["result"] = h.baseURL + shortURL
	c.IndentedJSON(http.StatusOK, "Success Register")
	h.HandlerLogin(c)
}
func (h *Handler) HandlerLogin(c *gin.Context) {
	var results string
	//POST /api/user/login HTTP/1.1
	//Content-Type: application/json
	//...
	//
	//{
	//	"login": "<login>",
	//	"password": "<password>"
	//}
	//Возможные коды ответа:
	//200 — пользователь успешно аутентифицирован;
	//400 — неверный формат запроса;
	//401 — неверная пара логин/пароль;
	//500 — внутренняя ошибка сервера.

	value := user{}
	defer c.Request.Body.Close()

	body, err := ioutil.ReadAll(c.Request.Body)

	if err := json.Unmarshal([]byte(body), &value); err != nil {
		c.IndentedJSON(http.StatusInternalServerError, "Server Error")
	}

	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, "Server Error")
		return
	}
	if (value.Login == "") || (value.Password == "") {
		c.IndentedJSON(http.StatusBadRequest, "Error")
		return
	}
	results, err = h.repo.Login(value.Login, value.Password, c)
	log.Println(results)
	if err != nil {
		var ue *DBError
		//if errors.As(err, &ue) && ue.Title == "user not found" {
		//	c.IndentedJSON(http.StatusConflict, "Status Conflict")
		//	return
		//}
		if errors.As(err, &ue) && (ue.Title == "wrong password" || ue.Title == "user not found") {
			c.IndentedJSON(http.StatusUnauthorized, "Status Unauthorized")
			return
		}
		c.IndentedJSON(http.StatusInternalServerError, err)
		return
	}
	c.IndentedJSON(http.StatusOK, "Success Login")
}

func (h *Handler) HandlerPostOrders(c *gin.Context) {
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
	//400 — неверный формат запроса;
	//401 — пользователь не аутентифицирован;
	//409 — номер заказа уже был загружен другим пользователем;
	//422 — неверный формат номера заказа;
	//500 — внутренняя ошибка сервера.
}

func (h *Handler) HandlerGetOrders(c *gin.Context) {
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
