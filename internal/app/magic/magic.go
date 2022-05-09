package magic

import "strings"

//Вызывается, после успешной регистрации
func Magic(order string) (string, error) {

	//Сделать магию
	res := strings.TrimLeft(order, "0")
	magicNum := len(res) / 10

	return string(magicNum), nil
}
