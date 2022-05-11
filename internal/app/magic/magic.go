package magic

import (
	"context"
	"encoding/json"
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
	log.Println("START AccrualAskWorker")
	c := time.Tick(time.Millisecond)
	for range c {
		go h.AccrualAskWorkerRunner()
	}
}
func (h *Handler) AccrualAskWorkerRunner() {
	log.Println("START AccrualAskWorkerRunner")
	c := context.Background()
	order, err := h.repo.GetNewOrder(c)
	if err != nil {
		//c.IndentedJSON(http.StatusInternalServerError, "Server Error")
		log.Println("Server Error  125")
		log.Println(err)
		return
	}
	log.Println("ORDER in Worker is ", order)

	if order != "" {

		log.Println("start celculate things order: ", order)

		//Принять заказ и изменить статус на "в обработке"
		value := ResponseAccural{}
		url := "http://" + h.accuralURL + "/api/orders/" + order
		log.Println("URL:")
		log.Println(url)

		response, err := http.Get(url) //
		if err != nil {
			//c.IndentedJSON(http.StatusInternalServerError, "Server Error")
			log.Println("Server Error http.Get(url)")
			log.Println(err)
			return
		}
		defer response.Body.Close()
		if response.StatusCode == 200 {
			body, err := ioutil.ReadAll(response.Body)
			if err != nil {
				//c.IndentedJSON(http.StatusInternalServerError, "Server Error")
				log.Println("Server Error ReadAll")
				log.Println(err)
				return
			}

			err = json.Unmarshal(body, &value)
			log.Println("body: ", body)
			log.Println("status is:", &value.Status)
			log.Println("accrual is:", &value.Accrual)
			log.Println("order is:", &value.Order)

			if value.Status == "REGISTERED" {
				log.Println("UpdateStatus(order, \"REGISTERED\"")
				return
			}
			if value.Status == "PROCESSING" {
				log.Println("UpdateStatus(order, \"PROCESSING\"")
				h.repo.UpdateStatus(order, "PROCESSING", c)
				return
			}

			if value.Status == "INVALID" {
				log.Println("value.Status == \"INVALID\"")
				h.repo.UpdateStatus(order, "INVALID", c)
				log.Println("UpdateStatus(order, \"INVALID\"")
				return
			}

			if value.Status == "PROCESSED" {
				log.Println("Start Update Wallet")
				err = h.repo.UpdateWallet(order, float32(value.Accrual), c)
				if err != nil {
					//h.repo.UpdateStatus(order, "INVALID", c)
					log.Println("UpdateWallet err")
					log.Println(err)
					return
				}
				//Изменить Accural
				//s := fmt.Sprintf("%f", value.Accrual)
				err = h.repo.UpdateAccural(order, float32(value.Accrual), c)
				log.Println("UpdateAccural")
				if err != nil {
					//h.repo.UpdateStatus(order, "INVALID", c)
					log.Println("\"UpdateAccural\" err")
					log.Println(err)
					return
				}
				log.Println("UpdateStatus(order, \"PROCESSED\"")
				err = h.repo.UpdateStatus(order, "PROCESSED", c)
				if err != nil {
					//h.repo.UpdateStatus(order, "INVALID", c)
					log.Println("\"UpdateStatus\" err")
					log.Println(err)
					return
				}
				//Начислить баллы

			}
		}

	}
}
