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

type MagicHandler struct {
	repo          handlers.MarketInterface
	serverAddress string
	accuralURL    string
	wp            workers.Workers
}

func New(repo handlers.MarketInterface, serverAddress string, accrualURL string, wp *workers.Workers) *MagicHandler {
	return &MagicHandler{
		repo:          repo,
		serverAddress: serverAddress,
		accuralURL:    accrualURL,
		wp:            *wp,
	}
}

type ResponseonseAccrual struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float32 `json:"accrual"`
}

var iteration = 0

func (h *MagicHandler) AccrualAskWorker() {

	c := time.Tick(time.Second)
	for range c {
		go h.AccrualAskWorkerRunner()
	}
}

func (h *MagicHandler) AccrualAskWorkerRunner() {

	log.Println("START AccrualAskWorkerRunner")
	c := context.Background()
	order, err := h.repo.GetNewOrder(c)
	if err != nil {
		//c.IndentedJSON(http.StatusInternalServerError, "Server Error")
		log.Println("Server Error  125")
		log.Println(err)
		return
	}

	response, err := http.Get(h.accuralURL + "/api/orders/" + order)
	if err != nil {
		log.Println(err)
		return
	}
	defer response.Body.Close()
	log.Println(response.StatusCode)
	if response.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Println(err)
		}

		var value ResponseonseAccrual
		if err := json.Unmarshal(body, &value); err != nil {
			log.Println(err)
		}
		fmt.Printf("%+v \n", value)

		fmt.Printf("%+v \n", value)
		log.Println("body: ", body)
		log.Println("status is:", value.Status)
		log.Println("accrual is:", value.Accrual)
		log.Println("order is:", value.Order)

		if value.Status == "REGISTERED" {
			log.Println("UpdateStatus(order, \"REGISTERED\"")
			return
		}
		if value.Status == "PROCESSING" {
			log.Println("UpdateStatus(order, \"PROCESSING\"")
			h.repo.UpdateStatus(c, order, "PROCESSING")
			return
		}

		if value.Status == "INVALID" {
			log.Println("value.Status == \"INVALID\"")
			h.repo.UpdateStatus(c, order, "INVALID")
			log.Println("UpdateStatus(order, \"INVALID\"")
			return
		}

		if value.Status == "PROCESSED" {
			log.Println("Start Update Wallet")
			newfloat := &value.Accrual
			log.Println("new float ", *newfloat)
			newnewfloat := value.Accrual
			log.Println("new new float ", newnewfloat)
			err = h.repo.UpdateWallet(c, order, value.Accrual)
			if err != nil {
				//h.repo.UpdateStatus(order, "INVALID", c)
				log.Println("UpdateWallet err")
				log.Println(err)
				return
			}
			//???????????????? Accural
			//s := fmt.Sprintf("%f", value.Accrual)
			err = h.repo.UpdateAccural(c, order, float32(value.Accrual))
			log.Println("UpdateAccural")
			if err != nil {
				//h.repo.UpdateStatus(order, "INVALID", c)
				log.Println("\"UpdateAccural\" err")
				log.Println(err)
				return
			}
			log.Println("UpdateStatus(order, \"PROCESSED\"")
			err = h.repo.UpdateStatus(c, order, "PROCESSED")
			if err != nil {
				//h.repo.UpdateStatus(order, "INVALID", c)
				log.Println("\"UpdateStatus\" err")
				log.Println(err)
				return
			}
		}
	}
}
