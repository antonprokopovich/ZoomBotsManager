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

	defaultRole = 0
)

var (
	dataFilename = flag.String("data", "users.json", "user data file name")
)

type userRecord struct {
	MeetNum  string
	MeetPass string
	Name     string
	Email    string
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
	if users, err := getUsersData(*dataFilename); err == nil {
		for _, user := range users {
			// Headless
			ctx, cancel := chromedp.NewContext(
				context.Background(),
				//chromedp.WithDebugf(log.Printf),
			)
			joinMeeting(ctx, cancel, user)

			time.Sleep(2 * time.Second)

		}
		waitFinish()
	}
}
