package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	fmtogram "github.com/iamissahar/Fmtogram"
	_ "github.com/lib/pq"
)

var (
	db          *sql.DB
	cachedUsers = map[int]struct{}{}
)

type t struct {
	name string
	id   int
}

func getReqAndResp(get fmtogram.IGet) {
	log.Printf("Request: %s", get.Request())
	log.Printf("Response: %s", get.Response())
}

func prepareGif(msg *fmtogram.Message, lang string, textcode int) {
	var err error
	anim := fmtogram.NewAnimation()
	if err = anim.WriteAnimationInternet(higif); err != nil {
		log.Printf("prepareGif (1): %s", err)
	}
	if err = msg.WriteAnimation(anim); err != nil {
		log.Printf("prepareGif (2): %s", err)
	}
}

func prepareText(msg *fmtogram.Message, lang string, textcode int) {
	var err error
	if err = msg.WriteString(text[lang][textcode]); err != nil {
		log.Printf("prepareText (1): %s", err)
	}
	if err = msg.WriteParseMode(fmtogram.HTML); err != nil {
		log.Printf("prepareText (2): %s", err)
	}
}

func addNewUser(userID int, lang string) error {
	var (
		res int
		err error
	)
	row := db.QueryRow("SELECT COUNT(*) FROM Users WHERE id = $1", userID)
	if err = row.Scan(&res); err != nil {
		log.Printf("addNewUser (1): %s", err)
	} else if res == 0 {
		_, err = db.Exec("INSERT INTO Users(id, lang) VALUES ($1, $2)", userID, lang)
		if err != nil {
			log.Printf("addNewUser (2): %s", err)
		}
	}
	return err
}

func greetings(bs *fmtogram.BasicSettings, lang string, userID int) error {
	var (
		actions   = []func(*fmtogram.Message, string, int){prepareText, prepareGif, prepareText}
		textcodes = []int{hello1, 0, hello2}
		err       error
	)
	if _, ok := cachedUsers[userID]; !ok {
		err = addNewUser(userID, lang)
		if err == nil {
			cachedUsers[userID] = struct{}{}
		}
	}
	if err == nil {
		for i, f := range actions {
			msg := fmtogram.NewMessage()
			get := msg.GetResults()
			f(msg, lang, textcodes[i])
			if err = bs.Send(msg, userID); err != nil {
				log.Printf("greetings: %s", err)
			}
			getReqAndResp(get)
		}
	}
	return err
}

func taskExists(taskID string, userID int) (bool, error) {
	var (
		amount int
		res    bool
		err    error
	)
	row := db.QueryRow(`
		SELECT COUNT(*) 
		FROM Relations
		JOIN Tasks ON Relations.task_id = Tasks.id
		WHERE Relations.user_id = $1
			AND Relations.task_id = $2
			AND Tasks.done = 0`, userID, taskID)
	if err = row.Scan(&amount); err != nil {
		log.Printf("taskExists: %s", err)
	} else if amount > 0 {
		res = true
	}
	return res, err
}

func updateDB(taskID string, userID int) error {
	var amount int
	_, err := db.Exec(`
			UPDATE Tasks 
			SET done = 1 
			FROM Relations 
			WHERE Tasks.id = Relations.task_id 
				AND Tasks.id = $1 
				AND Relations.user_id = $2`, taskID, userID)
	if err != nil {
		log.Printf("updateDB (1): %s", err)
	}
	err = db.QueryRow("SELECT allowed_amount_of_letters FROM Users WHERE id = $1", userID).Scan(&amount)
	if err != nil {
		log.Printf("updateDB (2): %s", err)
	} else {
		if amount == 10 {
			amount = 25
		} else if amount < 100 {
			amount += 25
		}
		_, err = db.Exec(`
			UPDATE Users
			SET allowed_amount_of_letters = $1
			WHERE id = $2`, amount, userID)
		if err != nil {
			log.Printf("updateDB (3): %s", err)
		}
	}
	return err
}

func taskIsDone(bs *fmtogram.BasicSettings, lang, taskID string, userID int) error {
	updateDB(taskID, userID)
	msg := fmtogram.NewMessage()
	get := msg.GetResults()
	tasks, err := getUserTasks(userID)
	if err == nil {
		if len(tasks) > 0 {
			anim := fmtogram.NewAnimation()
			if err = anim.WriteAnimationInternet(trumpwining[rand.Intn(len(trumpwining))]); err != nil {
				log.Printf("taskIsDone (1): %s", err)
			}
			if err = msg.WriteAnimation(anim); err != nil {
				log.Printf("taskIsDone (2): %s", err)
			}
			if err = msg.WriteString(text[lang][success]); err != nil {
				log.Printf("taskIsDone (3): %s", err)
			}
			if err = msg.WriteParseMode(fmtogram.HTML); err != nil {
				log.Printf("taskIsDone (4): %s", err)
			}
			if err = bs.Send(msg, userID); err != nil {
				log.Printf("taskIsDone (5): %s", err)
			}
			getReqAndResp(get)

			if err = createForm(bs, fmtogram.NewMessage(), text[lang][taskadded], tasks, userID); err != nil {
				log.Printf("taskIsDone (6): %s", err)
			}
		} else {
			anim := fmtogram.NewAnimation()
			if err = anim.WriteAnimationInternet(trumpwining[rand.Intn(len(trumpwining))]); err != nil {
				log.Printf("taskIsDone (7): %s", err)
			}
			if err = msg.WriteAnimation(anim); err != nil {
				log.Printf("taskIsDone (8): %s", err)
			}
			if err = bs.Send(msg, userID); err != nil {
				log.Printf("taskIsDone (9): %s", err)
			}
			getReqAndResp(get)

			msg = fmtogram.NewMessage()
			get = msg.GetResults()
			if err = msg.WriteString(text[lang][congrat]); err != nil {
				log.Printf("taskIsDone (10): %s", err)
			}
			if err = msg.WriteParseMode(fmtogram.HTML); err != nil {
				log.Printf("taskIsDone (11): %s", err)
			}
			if err = bs.Send(msg, userID); err != nil {
				log.Printf("taskIsDone (12): %s", err)
			}
			getReqAndResp(get)
		}
	}
	return err
}

func addTask(taskname string, userID int) (int, error) {
	var (
		taskID int
		err    error
	)
	row := db.QueryRow("SELECT nextval('tasks_id_seq')")
	if err = row.Scan(&taskID); err == nil {
		_, err = db.Exec("INSERT INTO Tasks (id, name) VALUES ($1, $2)", taskID, taskname)
		if err == nil {
			_, err = db.Exec("INSERT INTO Relations (task_id, user_id) VALUES ($1, $2)", taskID, userID)
		}
	}
	return taskID, err
}

func getUserTasks(userID int) ([]t, error) {
	var (
		res []t
		err error
	)
	rows, err := db.Query(`
		SELECT Tasks.id, Tasks.name 
		FROM Tasks
		INNER JOIN Relations ON Tasks.id = Relations.task_id 
		WHERE Relations.user_id = $1
		AND Tasks.done = 0 
		GROUP BY Tasks.id, Tasks.name`, userID)
	if err == nil {
		for rows.Next() {
			taskID, taskname := 0, ""
			if err = rows.Scan(&taskID, &taskname); err != nil {
				log.Printf("getUserTasks: %s", err)
			}
			res = append(res, t{taskname, taskID})
		}
	}
	return res, err
}

func isLengthOK(taskname string, userID int) bool {
	var (
		amount int
		res    bool
	)
	length := len([]rune(taskname))
	err := db.QueryRow("SELECT allowed_amount_of_letters FROM Users WHERE id = $1", userID).Scan(&amount)
	if err != nil {
		log.Printf("isLengthOK: %s", err)
	}
	if length < amount {
		res = true
	}
	return res
}

func createForm(bs *fmtogram.BasicSettings, msg *fmtogram.Message, strToAddTo string, tasks []t, userID int) error {
	var (
		str        string
		setbuttons []int
		err        error
		in         fmtogram.IInline
		btn        fmtogram.IInlineButton
	)
	get := msg.GetResults()
	for i, task := range tasks {
		str += fmt.Sprintf("<strong>%s %s</strong>\n", numbers[i], task.name)
		setbuttons = append(setbuttons, 1)
	}
	if err = msg.WriteString(fmt.Sprintf(strToAddTo, str+"\n")); err != nil {
		log.Printf("createForm (1): %s", err)
	}
	if err = msg.WriteParseMode(fmtogram.HTML); err != nil {
		log.Printf("createForm (2): %s", err)
	}
	kb := fmtogram.NewKeyboard()
	in, err = kb.WriteInline()
	if err != nil {
		log.Printf("createForm (3): %s", err)
	} else {
		if err = in.Set(setbuttons); err != nil {
			log.Printf("createForm (4): %s", err)
		}
		for i := range setbuttons {
			btn, err = in.NewButton(i, 0)
			if err != nil {
				log.Printf("createForm (5): %s", err)
			} else {
				if err = btn.WriteString(tasks[i].name); err != nil {
					log.Printf("createForm (6): %s", err)
				}
				if err = btn.WriteCallbackData(fmt.Sprint(tasks[i].id)); err != nil {
					log.Printf("createForm (7): %s", err)
				}
			}
		}
		if err = msg.WriteKeyboard(kb); err != nil {
			log.Printf("createForm (8): %s", err)
		}
		if err = bs.Send(msg, userID); err != nil {
			log.Printf("createForm (9): %s", err)
		}
		getReqAndResp(get)
	}
	return err
}

func getAllowedAmount(userID int) int {
	var amount int
	err := db.QueryRow("SELECT allowed_amount_of_letters FROM Users WHERE id = $1", userID).Scan(&amount)
	if err != nil {
		log.Printf("getAllowedAmount (1): %s", err)
	}
	return amount
}

func outOfTheLaw(bs *fmtogram.BasicSettings, lang string, userID int) {
	var err error
	msg := fmtogram.NewMessage()
	get := msg.GetResults()
	allowed := getAllowedAmount(userID)
	if err = msg.WriteString(fmt.Sprintf(text[lang][illigal], letters-allowed, allowed)); err != nil {
		log.Printf("outOfTheLaw (1): %s", err)
	}
	if err = msg.WriteParseMode(fmtogram.HTML); err != nil {
		log.Printf("outOfTheLaw (2): %s", err)
	}
	if err = bs.Send(msg, userID); err != nil {
		log.Printf("outOfTheLaw (3): %s", err)
	}
	getReqAndResp(get)
}

func tooManyTasks(bs *fmtogram.BasicSettings, lang string, tasks []t, userID int) error {
	var err error
	msg := fmtogram.NewMessage()
	get := msg.GetResults()
	anim := fmtogram.NewAnimation()
	if err = anim.WriteAnimationInternet(trumpnonono[rand.Intn(len(trumpnonono))]); err != nil {
		log.Printf("tooManyTasks (1): %s", err)
	}
	if err = msg.WriteAnimation(anim); err != nil {
		log.Printf("tooManyTasks (2): %s", err)
	}
	if err = msg.WriteString(text[lang][angry]); err != nil {
		log.Printf("tooManyTasks (3): %s", err)
	}
	if err = msg.WriteParseMode(fmtogram.HTML); err != nil {
		log.Printf("tooManyTasks (4): %s", err)
	}
	if err = bs.Send(msg, userID); err != nil {
		log.Printf("tooManyTasks (5): %s", err)
	}
	getReqAndResp(get)

	if err = createForm(bs, fmtogram.NewMessage(), text[lang][taskadded], tasks, userID); err != nil {
		log.Printf("tooManyTasks (6): %s", err)
	}

	return err
}

func botlogic(upd *fmtogram.Update, bs *fmtogram.BasicSettings) {
	var (
		taskID, userID int
		lang           string
		ok             bool
		err            error
		tasks          []t
	)
	if upd.Message != nil {
		userID = upd.Message.From.ID
		lang = upd.Message.From.LanguageCode
		if upd.Message.Text == "/start" {
			greetings(bs, lang, userID)
		} else {
			if tasks, err = getUserTasks(userID); len(tasks) < 4 && err == nil {
				if isLengthOK(upd.Message.Text, userID) {
					taskID, err = addTask(upd.Message.Text, userID)
					if err == nil {
						tasks = append(tasks, t{upd.Message.Text, taskID})
						createForm(bs, fmtogram.NewMessage(), text[lang][taskadded], tasks, userID)
					}
				} else {
					outOfTheLaw(bs, lang, userID)
				}
			} else {
				tooManyTasks(bs, lang, tasks, userID)
			}
		}
	} else if upd.CallbackQ != nil {
		userID = upd.CallbackQ.From.ID
		lang = upd.CallbackQ.From.LanguageCode
		if ok, err = taskExists(upd.CallbackQ.Data, userID); ok && err == nil {
			taskIsDone(bs, lang, upd.CallbackQ.Data, userID)
		}
	}
}

func updateTariffs(userID int) int {
	var err error
	amount := getAllowedAmount(userID)
	if amount <= 25 {
		amount = 10
	} else {
		amount -= 25
	}
	_, err = db.Exec("UPDATE Users SET allowed_amount_of_letters = $1 WHERE id = $2", amount, userID)
	if err != nil {
		log.Printf("updateTariffs (2): %s", err)
	}
	return amount
}

func setTimer(bs *fmtogram.BasicSettings) {
	for {
		var tasks []t
		var get fmtogram.IGet
		var msg *fmtogram.Message
		rows, err := db.Query(`
		SELECT Relations.user_id, Users.lang, Tasks.name, Tasks.id
		FROM Tasks
		JOIN Relations ON Tasks.id = Relations.task_id
		JOIN Users ON Relations.user_id = Users.id
		WHERE Tasks.done = 0
		AND Tasks.start_time < CURRENT_TIMESTAMP - INTERVAL '12 hours'`)
		if err == nil {
			for rows.Next() {
				userID, lang, name, taskID := 0, "", "", 0
				err = rows.Scan(&userID, &lang, &name, &taskID)
				if err == nil && userID != 0 {
					msg = fmtogram.NewMessage()
					get = msg.GetResults()
					anim := fmtogram.NewAnimation()
					anim.WriteAnimationInternet(trumpfalure[rand.Intn(len(trumpfalure))])
					msg.WriteAnimation(anim)
					bs.Send(msg, userID)
					getReqAndResp(get)

					msg = fmtogram.NewMessage()
					get = msg.GetResults()
					msg.WriteString(text[lang][fail])
					msg.WriteParseMode(fmtogram.HTML)
					bs.Send(msg, userID)
					getReqAndResp(get)

					allowed := updateTariffs(userID)

					msg = fmtogram.NewMessage()
					get = msg.GetResults()
					msg.WriteString(fmt.Sprintf(text[lang][illigal], letters-allowed, allowed))
					msg.WriteParseMode(fmtogram.HTML)
					bs.Send(msg, userID)
					getReqAndResp(get)

					_, err = db.Exec("UPDATE Tasks SET done = 2 WHERE id = $1", taskID)
					if err == nil {
						tasks, err = getUserTasks(userID)
						if err == nil {
							if len(tasks) > 0 {
								createForm(bs, fmtogram.NewMessage(), text[lang][taskadded], tasks, userID)
							} else {
								msg = fmtogram.NewMessage()
								get = msg.GetResults()
								msg.WriteString(text[lang][oops])
								msg.WriteParseMode(fmtogram.HTML)
								bs.Send(msg, userID)
								getReqAndResp(get)
							}
						}
					}
				}
			}
		}
		time.Sleep(time.Minute * 1)
	}
}

func handler(h chan error) {
	for err := range h {
		log.Print(err)
	}
}

func init() {
	var err error
	psqlconn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("host_db"), os.Getenv("port_db"), os.Getenv("user_db"),
		os.Getenv("password_db"), os.Getenv("dbname_db"), os.Getenv("sslmode_db"))
	db, err = sql.Open("postgres", psqlconn)
	if err != nil {
		panic(err)
	}

	if err = db.Ping(); err != nil {
		panic(err)
	}
}

func main() {
	t := 5
	tt := 100
	errorhandler := make(chan error)
	bs := &fmtogram.BasicSettings{
		StartFunc:      botlogic,
		Token:          botid,
		AllowedUpdates: []string{"message", "callback_query"},
		ErrorHandler:   errorhandler,
		Timeout:        &t,
		Limit:          &tt,
	}
	bs.Start()
	go setTimer(bs)
	handler(errorhandler)
}
