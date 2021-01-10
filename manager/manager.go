package main

import (
	"context"
	"fmt"
	"github.com/chromedp/chromedp-0.6.0"
	"log"
	"time"
)

// TODO get actual user data from config
const setUserData = `window.meetingNumber = "78563092610";
					 window.meetingPassword = "vF2Hk8";
					 window.meetingRole = 0;
					 window.leaveUrl = "http://localhost:3000";
					 window.userName = "Go Bot";
					 window.userEmail = "john@example.com";`


func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// Запускает браузер в хэдлесс-режиме
func initHeadless() (context.Context, context.CancelFunc) {
	ctx, cancel := chromedp.NewContext(
			context.Background(),
			chromedp.WithDebugf(log.Printf),
		)
	return ctx, cancel
}

// Запускает браузер в стандартном режиме (открывает окно)
func initNonHeadless() (context.Context, context.CancelFunc) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		// Set the headless flag to false to display the browser window
		chromedp.Flag("headless", false),
	)

	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(
		ctx,
		chromedp.WithDebugf(log.Printf),
	)
	return ctx, cancel
}

func setMeetingParamsTsk(evalCode string) chromedp.Tasks {
	var res []byte
	return chromedp.Tasks{
		chromedp.Evaluate(evalCode, &res),
	}
}

func clickJoinBtnTsk() chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.WaitVisible(`join-meeting-button`, chromedp.ByID),
		chromedp.Click(`join-meeting-button`, chromedp.ByID),
		chromedp.Sleep(5 * time.Second),
	}
}

func joinMeeting(setUserData string, ctxt context.Context) {
	checkErr(chromedp.Run(ctxt,
		chromedp.Navigate(`http://localhost:3000`),
		setMeetingParamsTsk(setUserData),
		clickJoinBtnTsk(),
	))
}

// Сохраняет подключение к митингу пока ведущий не завершит его
func waitFinish() {
	for {
		fmt.Println("Waiting for meeting to finish...")
		time.Sleep(60 * time.Second)

		//TODO получать сигнал о завершении
	}
}


func main() {
	// Headless
	//ctx, cancel := chromedp.NewContext(
	//	context.Background(),
	//	chromedp.WithDebugf(log.Printf),
	//)
	//defer cancel()

	// Non-headless
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		// Set the headless flag to false to display the browser window
		chromedp.Flag("headless", false),
	)

	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(
		ctx,
		//chromedp.WithDebugf(log.Printf),
	)
	defer cancel()

	joinMeeting(setUserData, ctx)

	waitFinish()

}