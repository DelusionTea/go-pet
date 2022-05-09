package magic

import (
	"log"
	"strings"
	"time"
)

//Вызывается, после успешной регистрации
func Magic(order string) (string, error) {
	log.Println("Call Magic func - it's pretty. Order is: ", order)
	//Сделать магию

	time.Sleep(1 * time.Second)
	res := strings.TrimLeft(order, "8")
	magicNum := len(res) / 10
	log.Println("End ofMagic func - it's pretty. Num is: ", magicNum)
	return string(magicNum), nil
}
