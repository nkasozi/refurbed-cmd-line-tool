/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"bufio"
	"fmt"
	async_http "github.com/nkasozi/refurbed-async-http-lib/http"
	"github.com/nkasozi/refurbed-async-http-lib/http/models"
	"github.com/spf13/cobra"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

var notifyUrl = ""
var notifyInterval time.Duration
var httpMethod = ""
var maxNumOfThreads = 3
var signals chan os.Signal

// notifyCmd represents the notify command
var notifyCmd = &cobra.Command{
	Use:   "notify",
	Short: "notify command - will send message(s) to supplied url at a rate of the supplied interval",
	Long: `notify command - is a CLI tool that helps you the supplied message(s) to supplied url at a rate of the supplied interval
 e.g go run main.go notify --url=testURL --interval=5s < messages.txt`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("notify called with notifyUrl:[%v] and notifyInterval:[%v], args:[%v]\n", notifyUrl, notifyInterval, args)

		asyncHttpRequestSender, osTerminateGlobalSignalChannel, pendingMessagesChannel, consoleInputProcessingDoneChannel := setUp()

		go listenAndProcessPendingMessagesAsync(asyncHttpRequestSender, pendingMessagesChannel, notifyInterval, consoleInputProcessingDoneChannel)

		go readAndProcessCmdLineInput(pendingMessagesChannel, osTerminateGlobalSignalChannel, args)

		checkForProgramTerminationSignals(consoleInputProcessingDoneChannel, osTerminateGlobalSignalChannel)
	},
}

func setUp() (asyncHttpRequestSender async_http.AsyncHttpRequestSender, osTerminateGlobalSignalChannel chan bool, pendingMessagesChannel chan string, consoleInputProcessingDoneChannel chan bool) {
	httpRequestSender := async_http.NewHttpRequestSender(http.Client{})
	asyncHttpRequestSender = async_http.NewAsyncHttpRequestsSender(httpRequestSender, maxNumOfThreads)

	osTerminateGlobalSignalChannel = make(chan bool, 1)
	pendingMessagesChannel = make(chan string, 1)
	consoleInputProcessingDoneChannel = make(chan bool, 1)
	return
}

func checkForProgramTerminationSignals(consoleInputProcessingDoneChannel <-chan bool, osTerminateGlobalSignalChannel chan<- bool) bool {
	for {

		select {

		//if we are done processing the input, we can terminate
		case _ = <-consoleInputProcessingDoneChannel:
			fmt.Println("Console Input Processing Is Done")
			return true

		//if the os signals for termination, we end
		case sig := <-signals:
			fmt.Println(sig)
			osTerminateGlobalSignalChannel <- true
			return true

		}

	}
}

func readAndProcessCmdLineInput(pendingMessagesChannel chan<- string, osTerminateGlobalSignalChannel <-chan bool, args []string) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		select {

		//if os has signalled for termination, we end
		case _ = <-osTerminateGlobalSignalChannel:
			close(pendingMessagesChannel)
			return

		default:

			//if message was passed as an argument
			if len(args) > 0 {

				for _, msg := range args {
					//we publish the msgs
					pendingMessagesChannel <- msg
				}

				close(pendingMessagesChannel)
				return
			}

			//otherwise read from console
			msg, shouldEnd := getSingleInput(scanner)

			//if console input is done, we terminate
			if shouldEnd {
				close(pendingMessagesChannel)
				return
			}

			//otherwise we publish the msg
			pendingMessagesChannel <- msg
		}
	}
}

func getSingleInput(scanner *bufio.Scanner) (msg string, shouldEnd bool) {
	// Scans a line from Stdin(Console)
	found := scanner.Scan()

	if !found {
		shouldEnd = true
		return
	}

	msg = scanner.Text()

	return msg, false
}

func listenAndProcessPendingMessagesAsync(asyncHttpRequestSender async_http.AsyncHttpRequestSender, pendingMessagesChannel <-chan string, notifyInterval time.Duration, consoleInputProcessingDoneChannel chan<- bool) {

	var wg sync.WaitGroup

	for {
		pendingMsg, isChannelOpen := <-pendingMessagesChannel

		// if there is nothing to do and the channel
		//has been closed then we just end
		if !isChannelOpen {
			fmt.Println("Channel Pending Messages has been closed")
			wg.Wait()
			fmt.Println("Responses for all Pending Messages have been received")
			consoleInputProcessingDoneChannel <- true
			return
		}

		wg.Add(1)

		fmt.Printf("received a Pending Message: [%v]\n", pendingMsg)

		//otherwise build and send the http message
		request := models.AsyncHttpRequest{
			Url:    notifyUrl,
			Body:   strings.NewReader(pendingMsg),
			Method: http.MethodPost,
			ResultHandler: func(resp *http.Response, err error) {

				if err != nil {
					fmt.Printf("Request Failed because of error: [%v]\n", err)
					return
				}

				if resp.StatusCode != http.StatusOK {
					fmt.Printf("Request Failed With Status Code: [%v]\n", resp.StatusCode)
					return
				}

				fmt.Printf("Recieved Success Response for Request\n")
				wg.Done()
			},
		}

		asyncHttpRequestSender.SendHttpRequestAsync(request)

		fmt.Printf("Successfully sent Pending Message: [%v]\n", pendingMsg)

		//wait for the configured duration
		time.Sleep(notifyInterval)
	}
}

func init() {
	rootCmd.AddCommand(notifyCmd)

	// Here we define our flags and configuration settings.
	notifyCmd.PersistentFlags().StringVarP(&notifyUrl, "url", "u", "", "Url to Notify")
	notifyCmd.PersistentFlags().StringVarP(&httpMethod, "method", "m", "POST", "Http Method to Use")
	notifyCmd.PersistentFlags().IntVarP(&maxNumOfThreads, "max-threads", "t", 3, "Maximum Number of Threads to use")
	notifyCmd.PersistentFlags().DurationVarP(&notifyInterval, "interval", "i", time.Duration(0), "Notification Interval")

	//mark flags that are required
	notifyCmd.MarkFlagRequired("url")
	notifyCmd.MarkFlagRequired("interval")

	initNotifyEnv()
}

func initNotifyEnv() {

	//initialize vars
	signals = make(chan os.Signal, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
}
