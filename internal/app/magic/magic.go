package magic

import (
	"log"
	"strings"
	"time"
)

//Вызывается, после успешной регистрации
func Magic(order string) (float64, error) {
	log.Println("Call Magic func - it's pretty. Order is: ", order)
	//Сделать магию
	var magicSolt = 5.4
	time.Sleep(1 * time.Second)
	res := strings.TrimLeft(order, "8")
	magicNum := magicSolt + float64(len(res))
	log.Println("End ofMagic func - it's pretty. Num is: ", magicNum)
	return magicNum, nil
}
