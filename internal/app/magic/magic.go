package magic

import (
	"strings"
	"time"
)

//Вызывается, после успешной регистрации
func Magic(order string) (string, error) {

	//Сделать магию
	time.Sleep(20 * time.Second)
	res := strings.TrimLeft(order, "0")
	magicNum := len(res) / 10

	return string(magicNum), nil
}
