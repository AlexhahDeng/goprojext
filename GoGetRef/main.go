package main

import (
	"fmt"

	"code.sajari.com/docconv"
)

func pdf2txt(filepath string) (string, error) {
	res, err := docconv.ConvertPath(filepath)
	if err != nil {
		return "nil", err
	}
	fmt.Println(res)
	return "res", nil
}
func main() {
	fmt.Println("hello jasmine")
	pdf2txt("he.pdf")
}
