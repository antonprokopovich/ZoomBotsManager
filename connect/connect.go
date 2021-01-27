package main

import (
	"context"
	"fmt"
	"github.com/chromedp/chromedp-0.6.0"
	"github.com/spf13/viper"
	"log"
	"time"
)

const (
	leaveUrl = `http://localhost:3000`

	defaultRole = 0
)

type conRecord struct {
	MeetNum    string
	MeetPass   string
	StartTime  string
	FinishTime string
	UserName   string
	UserEmail  string
}

type config struct {
	Connections []conRecord
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func makeCallString(data conRecord) string {
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
		data.UserName,
		data.UserEmail,
		leaveUrl,
	)
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

// Запускает браузер и переходит на страницу подключения к митингу
func navigateToPage(ctxt context.Context, url string) error {
	err := chromedp.Run(ctxt,
		chromedp.Navigate(url),
	)
	if err != nil {
		return err
	}
	return nil
}

// Осуществляет подключение пользователя к митингу на заданное время,
//выполняя задачи перехода на страницу клинта, установки параметров и
//нажатия кнопки
func joinMeeting(ctxt context.Context, con conRecord, to time.Duration) {
	ctxtTimed, cancel := context.WithTimeout(ctxt, to)
	defer cancel()

	callString := makeCallString(con)
	if err := navigateToPage(ctxtTimed, leaveUrl); err != nil {
		fmt.Println("Couldn't connect to " + leaveUrl)
		return
	}
	if err := chromedp.Run(ctxtTimed,
		setMeetingParamsTsk(callString),
		clickJoinBtnTsk(),
	); err != nil {
		fmt.Println("Couldn't joint the meeting #" + con.MeetNum)
		return
	}
}

// Сохраняет подключение к митингу пока ведущий не завершит его
func waitFinish() {
	for {
		fmt.Println("Waiting for meeting to finish...")
		time.Sleep(60 * time.Second)

		//TODO получать сигнал о завершении
	}
}

func getCfg() config {

	viper.SetConfigType("json")
	viper.AddConfigPath("./config")
	viper.SetConfigName("config")

	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	var cfg config
	err := viper.Unmarshal(&cfg)
	if err != nil {
		panic("Unable to unmarshal config")
	}

	return cfg
}

func main() {
	cfg := getCfg()
	for _, con := range cfg.Connections {
		func() {
			ctx, cancel := chromedp.NewContext(
				context.Background(),
				//chromedp.WithDebugf(log.Printf),
			)
			defer cancel()

			joinMeeting(ctx, con, 30*time.Second)
			time.Sleep(2 * time.Second)
		}()

	}
	waitFinish()
}
