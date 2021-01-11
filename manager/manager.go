package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/chromedp/chromedp-0.6.0"
	"log"
	"os"
	"time"
)

const (
	leaveUrl = `http://localhost:3000`

	test1 = `window.meetingNumber = "74768911708";
			 window.meetingPassword = "z6ti6C";
			 window.meetingRole = 0;
			 window.userName = "John Doe";
			 window.userEmail = "john@example.com";
			 window.leaveUrl = "http://localhost:3000";`

	test2 = `window.meetingNumber = "74768911708";
					 window.meetingPassword = "z6ti6C";
					 window.meetingRole = 0;
					 window.leaveUrl = "http://localhost:3000";
					 window.userName = "MJ";
					 window.userEmail = "john@example.com";`

	defaultRole = 0
)

var (
	dataFilename = flag.String("data", "users.json", "user data file name")
)

type userRecord struct {
	MeetNum   string
	MeetPass  string
	Name  string
	Email string
}


func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func makeCallString(data userRecord) string {
	return fmt.Sprintf(
		`window.meetingNumber = "%s";
				window.meetingPassword = "%s";
				window.meetingRole = %d;
				window.userName = "%s";
				window.userEmail = "%s";
				window.leaveUrl = "%s";`,
				data.MeetNum,
				data.MeetPass,
				defaultRole,
				data.Name,
				data.Email,
				leaveUrl,
	)
}

// Загружает (из json-файла) данные каждого пользователя в
// объект userRecord и возвращает массив с этими объектами
func getUsersData(filename string) ([]userRecord, error) {
	var usrs []userRecord

	f, err := os.Open(filename)
	if err != nil {
		fmt.Println("Error opening data file:", err)
		return nil, err
	}
	defer f.Close()

	d := json.NewDecoder(f)
	if err := d.Decode(&usrs); err != nil {
		fmt.Println("Error decoding data file:", err)
		return nil, err
	}
	return usrs, nil
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

// Возвращает задачу установки параметров для подключения
func setMeetingParamsTsk(callString string) chromedp.Tasks {
	var res []byte
	return chromedp.Tasks{
		chromedp.Evaluate(callString, &res),
	}
}

// Возвращает задачу нажатия кнопки подключения
func clickJoinBtnTsk() chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.WaitVisible(`join-meeting-button`, chromedp.ByID),
		chromedp.Click(`join-meeting-button`, chromedp.ByID),
		chromedp.Sleep(5 * time.Second),
	}
}

// Осуществляет подключение пользователя к митингу, выполняя задачи
// перехода на страницу клинта, установки параметров и нажатия кнопки
func joinMeeting(ctxt context.Context, cancel context.CancelFunc, user userRecord) {
	defer cancel()

	callString := makeCallString(user)

	//if callString != test1 {
	//	fmt.Println(callString)
	//	fmt.Println(test1)
	//	panic("!!! NOT EQUAL !!!: " )
	//}

	fmt.Println(callString)
	checkErr(chromedp.Run(ctxt,
		chromedp.Navigate(leaveUrl),
		setMeetingParamsTsk(callString),
		clickJoinBtnTsk(),
	))
	time.Sleep(2 * time.Second)
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
	// Non-headless
	//opts := append(chromedp.DefaultExecAllocatorOptions[:],
	//	chromedp.DisableGPU,
	//	// Set the headless flag to false to display the browser window
	//	chromedp.Flag("headless", false),
	//)
	//
	//ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	//defer cancel()
	//
	//ctx, cancel = chromedp.NewContext(
	//	ctx,
	//	//chromedp.WithDebugf(log.Printf),
	//)


	users, err := getUsersData(*dataFilename)
	if err == nil {
		for _, user := range users {

			// Headless
			ctx, cancel := chromedp.NewContext(
				context.Background(),
				//chromedp.WithDebugf(log.Printf),
			)
			joinMeeting(ctx, cancel, user)

			time.Sleep(2 * time.Second)

		}
	}

	waitFinish()

}

//test
func _main() {
	usr := userRecord{
		MeetNum:   "74768911708",
		MeetPass:  "z6ti6C",
		Name:  "John Doe",
		Email: "john@example.com",
	}
	//fmt.Println(makeCallString(usr))

	//data, err := getUsersData("users.json")
	//if err == nil {
	//	for _, u := range data {
	//		fmt.Println(u)
	//	}
	//}

	// Headless
	ctx, cancel := chromedp.NewContext(
		context.Background(),
		//chromedp.WithDebugf(log.Printf),
	)

	joinMeeting(ctx, cancel, usr)
	waitFinish()
}
