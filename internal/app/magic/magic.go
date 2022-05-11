package magic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/DelusionTea/go-pet.git/internal/app/handlers"
	"github.com/DelusionTea/go-pet.git/internal/workers"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type ResponseAccural struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float32 `json:"accrual"`
}

type Handler struct {
	repo          handlers.MarketInterface
	serverAddress string
	accuralURL    string
	wp            workers.Workers
}

func New(repo handlers.MarketInterface, serverAddress string, accrualURL string, wp *workers.Workers) *Handler {
	return &Handler{
		repo:          repo,
		serverAddress: serverAddress,
		accuralURL:    accrualURL,
		wp:            *wp,
	}
}

func (h *Handler) AccrualAskWorker() {

	c := time.Tick(time.Second)
	for range c {
		go h.AccrualAskWorkerRunner()
	}
}
func (h *Handler) AccrualAskWorkerRunner() {
	c := context.Background()
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

		//h.repo.UpdateStatus(order, "PROCESSING", c)
		//log.Println("UpdateStatus(order, \"PROCESSING\", c)", order)
		//log.Println("start Magic")

		//Начислить баллы
		log.Println("Start Update Wallet")
		err := h.repo.UpdateWallet(order, value.Accrual, c)
		if err != nil {
			h.repo.UpdateStatus(order, "INVALID", c)
			log.Println("UpdateStatus(order, \"INVALID\"")
			log.Println(err)
			return
		}
		//Изменить Accural
		s := fmt.Sprintf("%f", value.Accrual)
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
