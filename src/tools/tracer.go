/*
 * © 2018 SwiftyCloud OÜ. All rights reserved.
 * Info: info@swifty.cloud
 */

package main

import (
	"net"
	"encoding/json"
	"sort"
	"os"
	"fmt"
	"swifty/apis"
)

func tracerConnect(id, addr string) (*net.UnixConn, error) {
	ua, err := net.ResolveUnixAddr("unixpacket", addr)
	if err != nil {
		return nil, err
	}

	sk, err := net.DialUnix("unixpacket", nil, ua)
	if err != nil {
		return nil, err
	}

	hm := swyapi.TracerHello{ ID: id }
	data, _ := json.Marshal(&hm)
	_, err = sk.Write(data)
	if err != nil {
		sk.Close()
		return nil, err
	}

	return sk, nil
}

func main() {
	if len(os.Args) == 1 {
		fmt.Printf("Usage: %s <id> <socket-path>\n", os.Args[0])
		fmt.Printf("  <id> can be\n")
		fmt.Printf("       - 'ten:user-name' to watch events for a user\n")
		fmt.Printf("  <socket-path> is where gate keeps the listener\n")
		fmt.Printf("                likely this is /var/run/swifty/tracer.sock\n")
		return
	}

	fmt.Printf("Tracing reqs for %s (@%s)\n", os.Args[1], os.Args[2])

	sk, err := tracerConnect(os.Args[1], os.Args[2])
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		return
	}

	defer sk.Close()

	var prevr uint64
	prevr = 0

	msg := make([]byte, 1024)
	for {
		l, err := sk.Read(msg)
		if err != nil {
			fmt.Printf("Error reading from tracer: %s\n", err.Error())
			break
		}

		var tm swyapi.TracerEvent
		err = json.Unmarshal(msg[:l], &tm)
		if err != nil {
			fmt.Printf("Error parsing message: %s\n", err.Error())
			break
		}

		if tm.Type == "call" {
			fmt.Printf("%s %s.%s//%s//%s\n", tm.Ts.Format("15:04:05.000"),
					tm.Data["event"], tm.Data["method"], tm.Data["fname"], tm.Data["path"])
			fmt.Printf("\t%v (%v)\n", tm.Data["code"], tm.Data["status"])
			type x struct {
				n	string
				d	uint64
			}
			times := []x{}
			for n, dur := range tm.Data["times"].(map[string]interface{}) {
				times = append(times, x{n:n, d:uint64(dur.(float64))})
			}
			sort.Slice(times, func(i, j int) bool {
				return times[i].d < times[j].d
			})
			for _, t := range times {
				fmt.Printf("\t   %-10s%16d\n", t.n, t.d)
			}

			continue
		}

		var rqid string
		if tm.RqID == prevr {
			rqid = "      `-"
		} else {
			rqid = fmt.Sprintf("%08d", tm.RqID)
		}
		fmt.Printf("%s %s%6s:  ", tm.Ts.Format("15:04:05.000"), rqid, tm.Type)
		prevr = tm.RqID

		switch tm.Type {
		case "req":
			fmt.Printf("%s %s\n", tm.Data["method"], tm.Data["path"])
		case "resp":
			fmt.Printf("%s\n", tm.Data["values"])
		case "error":
			fmt.Printf("%d %s\n", tm.Data["code"], tm.Data["message"])
		default:
			fmt.Printf("%v\n", tm.Data)
		}
	}
}
