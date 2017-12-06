package main

import (
	"fmt"
	"swycode"
	"encoding/json"
	"xqueue"
	"os"
)

func main() {
	q, err := xqueue.OpenQueue(os.Args[1])
	if err != nil {
		fmt.Printf("Can't open queue %s: %s", os.Args[1], err.Error())
		return
	}

	for {
		var args map[string]string

		err = q.Recv(&args)
		if err != nil {
			fmt.Printf("Can't receive message: %s", err.Error())
			return
		}

		res := swifty.Main(args)

		var resb []byte
		resb, err = json.Marshal(res)
		if err != nil {
			fmt.Printf("Can't marshal the result: %s", err.Error())
			return
		}

		err = q.SendBytes(resb)
		if err != nil {
			fmt.Printf("Can't send responce: %s", err.Error())
			return
		}
	}
}
