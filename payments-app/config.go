package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
)

type Config struct {
	ProcessorUsername *string
	ProcessorPassword *string
	DatabaseUsername  *string
	DatabasePassword  *string
	DB                *sql.DB
}

func ScanDBConfig(file *os.File, config *Config) {

	scanner := bufio.NewScanner(file)
	// optionally, resize scanner's capacity for lines over 64K, see next example
	for scanner.Scan() {
		txt := scanner.Text()
		//fmt.Println(scanner.Text())
		if len(txt) > 0 {
			split := strings.Split(txt, "=")
			//fmt.Println("username: ", split)
			switch value := split[0]; value {
			case "username":
				fmt.Println("username: ", split[1])
				config.DatabaseUsername = &split[1]
			case "password":
				fmt.Println("password: ", split[1])
				config.DatabasePassword = &split[1]
			case "url":
				fmt.Println("url: ", split[1])
			}
		}

	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func ScanProcessorConfig(file *os.File, config *Config) {

	scanner := bufio.NewScanner(file)
	// optionally, resize scanner's capacity for lines over 64K, see next example
	for scanner.Scan() {
		txt := scanner.Text()
		//fmt.Println(scanner.Text())
		if len(txt) > 0 {
			split := strings.Split(txt, "=")
			//fmt.Println("username: ", split)
			switch value := split[0]; value {
			case "username":
				fmt.Println("username: ", split[1])
				config.ProcessorUsername = &split[1]
			case "password":
				fmt.Println("password: ", split[1])
				config.ProcessorPassword = &split[1]
			case "url":
				fmt.Println("url: ", split[1])
			}
		}

	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
