package main

import (
	"fmt"
	"net/http"
	"io/ioutil"
	"time"
)

func main(){
	for i:=0;i<10;i++ {
		go func(){
			resp, err := http.Get("http://127.0.0.1:1234")
			if err != nil {
				// handle error
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil{
				fmt.Println(err)
			}
			fmt.Println(string(body))
		}()
	}

	time.Sleep(3e9)
}
