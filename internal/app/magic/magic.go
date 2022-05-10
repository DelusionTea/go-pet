package magic

import (
	"log"
)

//Вызывается, после успешной регистрации
func Magic(order string) (float64, error) {
	//log.Println("Call Magic func - it's pretty. Order is: ", order)
	////Сделать магию
	//var magicSolt = 5.4
	//time.Sleep(1 * time.Second)
	//magicNum := magicSolt + float64(len(order))
	magicNum := 30.5
	log.Println("End ofMagic func - it's pretty. Num is: ", magicNum)
	return magicNum, nil
}
