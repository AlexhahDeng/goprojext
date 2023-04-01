package main

import (
	"fmt"
	"regexp"

	"code.sajari.com/docconv"
)

// convert pdf 2 txt
func pdf2txt(filepath string) (string, error) {
	res, err := docconv.ConvertPath("he.pdf")
	if err != nil {
		return "", err
	}
	return res.Body, err
}

// obtain reference
func txt2ref(txt string) (string, error) {
	r, _ := regexp.Compile("(?i)references")
	strIdx := r.FindStringIndex(txt)
	if strIdx == nil {
		return "", fmt.Errorf("no match reference")
	}
	fmt.Println("reference is at:", strIdx)
	return txt[strIdx[0]:], nil
}

// split ref into sub items
func splitRef(ref string) ([]string, error) {
	return nil, nil
}
func main() {
	txt, err := pdf2txt("he.pdf")
	if err != nil {
		fmt.Println("error in dealing with pdf")
		return
	}
	ref, err := txt2ref(txt)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(ref)

}
