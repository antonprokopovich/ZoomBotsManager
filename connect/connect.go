package main

import (
	"context"
	"fmt"
	"github.com/chromedp/chromedp-0.6.0"
	"github.com/spf13/viper"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	leaveUrl = `http://localhost:3000`

	defaultRole = 0
)

type conRecord struct {
	MeetNum   string
	MeetPass  string
	Date      string
	Time      string
	Duration  string
	UserName  string
	UserEmail string
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

// TODO parse to int (not int64)
func stringToInt(s string) (int, error) {
	value, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		fmt.Println("Couldn't parse time string")
		return 0, err
	}
	return int(value), nil
}

// Парсит строку времени начала конференции формата "HH:MM:SS" и возращает
// три целых цисла - час, минута, секунда
func parseStartTime(time string) (hour int, minute int, second int) {
	a := strings.Split(time, ":")

	h, err := stringToInt(a[0])
	if err != nil {
		fmt.Println(err)
		return
	}
	m, err := stringToInt(a[1])
	if err != nil {
		fmt.Println(err)
		return
	}
	s, err := stringToInt(a[2])
	if err != nil {
		fmt.Println(err)
		return
	}

	//fmt.Printf("Hour: %d, Minute: %d, Second: %d", h, m, s)
	return h, m, s
}

// TODO
func parseStartDate(date string) (day int, month int, year int) {
	return 0, 0, 0
}

func parseDuration(dur string) time.Duration {
	m, _ := stringToInt(dur)
	return time.Duration(m) * time.Minute
}

type jobTicker struct {
	timer *time.Timer
}

func (t *jobTicker) setTimerToday(hour int, minute int, second int) {
	nextTick := time.Date(time.Now().Year(), time.Now().Month(),
		time.Now().Day(), hour, minute, second, 0, time.Local)
	if !nextTick.After(time.Now()) {
		//nextTick = nextTick.Add(IntervalPeriod)
		panic("Can't join meeting in the past")
	}

	diff := nextTick.Sub(time.Now())
	if t.timer == nil {
		t.timer = time.NewTimer(diff)
	} else {
		t.timer.Reset(diff)
	}
}

//Осуществляет подключение пользователя к митингу на заданное время,
//выполняя задачи перехода на страницу клинта, установки параметров и
//нажатия кнопки
func joinMeeting(ctxtMain context.Context, cancelMain context.CancelFunc, conData conRecord, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		fmt.Printf(
			"Canceling main context for meeting: %s user: %s\n",
			conData.MeetNum, conData.UserName,
		)
		cancelMain()
	}()

	dur := parseDuration(conData.Duration)
	h, m, s := parseStartTime(conData.Time)

	fmt.Printf("Will join meeting %s at %d:%d:%d \n", conData.MeetNum, h, m, s)

	t := jobTicker{}
	t.setTimerToday(h, m, s)
	<-t.timer.C

	fmt.Printf("Joining meeting %s \n", conData.MeetNum)

	callString := makeCallString(conData)
	if err := navigateToPage(ctxtMain, leaveUrl); err != nil {
		fmt.Println("Couldn't connect to " + leaveUrl)
		fmt.Println(err)
		return
	}
	if err := chromedp.Run(ctxtMain,
		setMeetingParamsTsk(callString),
		clickJoinBtnTsk(),
		// Сохраняем подключение к митингу в течении заданного периода
		chromedp.Sleep(dur),
	); err != nil {
		fmt.Println("Couldn't joint the meeting #" + conData.MeetNum)
		return
	}
}

//Проходит по списку с данными подключений и
//подключает каждого человека к назначенной ему конференции в
//указанное время и на указанный период. По истечению периода
//отключает пользователя
func main() {

	cfg := getCfg()

	var wg sync.WaitGroup
	for _, con := range cfg.Connections {
		func() {
			ctx, cancel := chromedp.NewContext(
				context.Background(),
				//chromedp.WithDebugf(log.Printf),
			)

			wg.Add(1)
			go joinMeeting(ctx, cancel, con, &wg)
		}()
	}
	wg.Wait()
}
