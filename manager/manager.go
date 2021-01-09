package main

import (
	"context"
	"fmt"
	"github.com/chromedp/chromedp-0.6.0"
	"log"
	"time"
)

// TODO get actual user data from config?
const setUserData = `window.username = "John Doe";
				window.meetingNum = "77605673353";
				window.meetingPass = "1ei40S";`


func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func joinMeeting(setDataJS string, ctxt context.Context) {
	//var setDataRes []byte
	checkErr(chromedp.Run(ctxt,
		chromedp.Navigate(`http://localhost:3000`),
		//chromedp.Evaluate(setUserData, &setDataRes),
		chromedp.WaitVisible(`join-meeting-button`, chromedp.ByID),
		chromedp.Click(`join-meeting-button`, chromedp.ByID),
		chromedp.Sleep(5 * time.Second),

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
	ctxt, cancel := chromedp.NewContext(
			context.Background(),
			//chromedp.WithDebugf(log.Printf),
		)
	defer cancel()

	joinMeeting(setUserData, ctxt)

	waitFinish()

}