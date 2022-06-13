package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

type User struct {
	Browsers []string `json:"browsers"`
	Company  string   `json:"company"`
	Country  string   `json:"country"`
	Email    string   `json:"email"`
	Job      string   `json:"job"`
	Name     string   `json:"name"`
	Phone    string   `json:"phone"`
}

func FastSearch(out io.Writer) {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	fmt.Fprintln(out, "found users:")

	browsers := make(map[string]bool, 200)
	user := new(User)
	var isAndroid, isMSIE bool

	scan := bufio.NewScanner(file)
	for i := 0; scan.Scan(); i++ {
		isAndroid, isMSIE = false, false

		if err = json.Unmarshal(scan.Bytes(), user); err != nil {
			panic(err)
		}

		for _, browser := range user.Browsers {
			if strings.Contains(browser, "MSIE") {
				isMSIE = true
				browsers[browser] = true
			} else if strings.Contains(browser, "Android") {
				isAndroid = true
				browsers[browser] = true
			}
		}

		if isAndroid && isMSIE {
			email := strings.Replace(user.Email, "@", " [at] ", 1)
			fmt.Fprintln(out, fmt.Sprintf("[%d] %s <%s>", i, user.Name, email))
		}
	}

	fmt.Fprintln(out, "\nTotal unique browsers", len(browsers))
}
