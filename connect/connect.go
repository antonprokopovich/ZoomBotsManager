package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/chromedp/chromedp-0.6.0"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	leaveUrl = `http://localhost:3000`

	defaultRole = 0
)

type pendingConList struct {
	m    sync.RWMutex
	cons []string
}

type conRecord struct {
	MeetNum   string
	MeetPass  string
	Date      string
	Time      string
	Duration  string
	UserName  string
	UserEmail string
}

// Хэширует запись соединения.
// Используется для создания уникального ключа, под которым
// соединение будет храниться в памяти после прочтения его из конфига
func (c conRecord) asSha256() string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", c)))

	return fmt.Sprintf("%x", h.Sum(nil))
}

type config struct {
	Connections []conRecord
}

type timeData struct {
	hour   int
	minute int
	second int
}

type dateData struct {
	day   int
	month int
}

type jobTicker struct {
	timer *time.Timer
}

func (t *jobTicker) setTimer(dd dateData, td timeData) {
	nextTick := time.Date(
		time.Now().Year(), time.Month(dd.month), dd.day,
		td.hour, td.minute, td.second, 0, time.Local)

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

func isStartOutdated(r conRecord) bool {
	td, dd := parseStartTime(r.Time), parseStartDate(r.Date)

	start := time.Date(
		time.Now().Year(), time.Month(dd.month), dd.day,
		td.hour, td.minute, td.second, 0, time.Local)

	if start.After(time.Now()) {
		return false
	}

	return true
}

func NewPendingStore() *pendingConList {
	return &pendingConList{
		cons: make([]string, 0),
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
func parseStartTime(time string) (td timeData) {
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
	return timeData{h, m, s}
}

// Парсит строку даты начала конференции формата "DD:MM" и возращает
// три целых цисла - день, месяц
func parseStartDate(date string) (dd dateData) {
	a := strings.Split(date, ".")

	d, err := stringToInt(a[0])
	if err != nil {
		fmt.Println(err)
		return
	}
	m, err := stringToInt(a[1])
	if err != nil {
		fmt.Println(err)
		return
	}

	//fmt.Printf("Day: %d, Month: %d", d, m)
	return dateData{d, m}
}

func parseDuration(dur string) time.Duration {
	m, _ := stringToInt(dur)
	return time.Duration(m) * time.Minute
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
	sTime := parseStartTime(conData.Time)
	sDate := parseStartDate(conData.Date)

	fmt.Printf("Will join meeting %s at %d:%d:%d \n",
		conData.MeetNum, sTime.hour, sTime.minute, sTime.second)

	t := jobTicker{}
	t.setTimer(sDate, sTime)
	<-t.timer.C

	fmt.Printf("%s is joining meeting %s for %s minutes \n",
		conData.UserName, conData.MeetNum, conData.Duration)

	if err := navigateToPage(ctxtMain, leaveUrl); err != nil {
		fmt.Println("Couldn't connect to " + leaveUrl)
		fmt.Println(err)
		return
	}
	callString := makeCallString(conData)
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

func isPending(c conRecord) bool {
	pending.m.RLock()
	defer pending.m.RUnlock()

	for _, p := range pending.cons {
		if p == c.asSha256() {
			fmt.Printf("Connection is already pending (user: %s meeting: %s)",
				c.UserName, c.MeetNum)
			return true
		}
	}
	return false
}

func initNewCons(cfg config, wg *sync.WaitGroup) {
	for _, con := range cfg.Connections {
		if !isStartOutdated(con) && !isPending(con) {
			ctx, cancel := chromedp.NewContext(
				context.Background(),
				//chromedp.WithDebugf(log.Printf),
			)

			pending.m.Lock()
			pending.cons = append(pending.cons, con.asSha256())
			pending.m.Unlock()

			wg.Add(1)
			go joinMeeting(ctx, cancel, con, wg)
		}
	}
}

var pending *pendingConList

//Проходит по списку с данными подключений и
//подключает каждого человека к назначенной ему конференции в
//указанное время и на указанный период. По истечению периода
//отключает пользователя
func main() {
	pending = NewPendingStore()
	cfg := getCfg()

	var wg sync.WaitGroup

	viper.WatchConfig()
	// TODO запускать initNewCons по обновлению конфига
	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config file changed:", e.Name)
	})

	initNewCons(cfg, &wg)

	wg.Wait()
}
