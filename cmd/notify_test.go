package cmd

import (
	"github.com/golang/mock/gomock"
	"github.com/nkasozi/refurbed-async-http-lib/http/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"syscall"
	"testing"
	"time"
)

func TestReadAndProcessCmdLineInput(t *testing.T) {
	t.Run("given valid inputs messages, publishes messages to channel and closes channel after", func(t *testing.T) {

		osTerminateGlobalSignalChannel := make(chan bool, 1)
		pendingMessagesChannel := make(chan string, 1)

		msgFound := false

		go func() {
			for {
				select {
				case _, isOpen := <-pendingMessagesChannel:
					if !isOpen {
						return
					}
					msgFound = true
					return
				}
			}
		}()

		args := []string{"test message"}

		readAndProcessCmdLineInput(pendingMessagesChannel, osTerminateGlobalSignalChannel, args)

		time.Sleep(1 * time.Second)

		assert.True(t, msgFound)
	})

	t.Run("given valid os terminate message, closes channel", func(t *testing.T) {
		osTerminateGlobalSignalChannel := make(chan bool, 1)
		pendingMessagesChannel := make(chan string, 1)

		isOpen := true

		readAndProcessCmdLineInput(pendingMessagesChannel, osTerminateGlobalSignalChannel, []string{})

		osTerminateGlobalSignalChannel <- true

		select {
		case _, isOpen = <-pendingMessagesChannel:
			if !isOpen {
				return
			}

			return
		}

		assert.False(t, isOpen)
	})
}

func TestCheckForProgramTerminationSignals(t *testing.T) {
	t.Run("given a signal indicating console processing is done, returns true", func(t *testing.T) {
		osTerminateGlobalSignalChannel := make(chan bool, 1)
		consoleInputProcessingDoneChannel := make(chan bool, 1)

		go func() {
			for {
				consoleInputProcessingDoneChannel <- true
			}
		}()

		result := checkForProgramTerminationSignals(consoleInputProcessingDoneChannel, osTerminateGlobalSignalChannel)

		assert.True(t, result)
	})

	t.Run("given a signal indicating os signal SIGINT,publishes terminate signal and returns true", func(t *testing.T) {
		osTerminateGlobalSignalChannel := make(chan bool, 1)
		consoleInputProcessingDoneChannel := make(chan bool, 1)

		initNotifyEnv()

		msgFound := false

		go func() {
			for {
				select {
				case _, isOpen := <-osTerminateGlobalSignalChannel:
					if !isOpen {
						return
					}
					msgFound = true
					return
				}
			}
		}()

		go checkForProgramTerminationSignals(consoleInputProcessingDoneChannel, osTerminateGlobalSignalChannel)

		signals <- syscall.SIGINT

		time.Sleep(1 * time.Second)

		assert.True(t, msgFound)
	})
}

func TestListenAndProcessPendingMessagesAsync(t *testing.T) {
	t.Run("given a pending message, calls the right methods", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		pendingMessagesChannel := make(chan string, 1)
		consoleInputProcessingDoneChannel := make(chan bool, 1)
		asyncHttpRequestSender := mocks.NewMockAsyncHttpRequestSender(ctrl)
		asyncHttpRequestSender.EXPECT().SendHttpRequestAsync(mock.Anything).Return(nil).MinTimes(1)

		go listenAndProcessPendingMessagesAsync(asyncHttpRequestSender, pendingMessagesChannel, time.Duration(0), consoleInputProcessingDoneChannel)

		time.Sleep(1 * time.Second)

		pendingMessagesChannel <- "test message"
	})
}
