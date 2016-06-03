package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"golang.org/x/net/context"

	"google.golang.org/grpc"

	"github.com/dgraph-io/gru/server/interact"
	"github.com/gizak/termui"
)

var token = flag.String("token", "testtoken", "Authentication token")

const (
	// Test duration in minutes
	testDur = 60
	address = "localhost:8888"
)

// Elements for the questions page.
var instructions *termui.Par
var timeLeft *termui.Par
var que *termui.Par
var score *termui.Par
var s *termui.Par
var a *termui.Par

// Elements for the home page.
var demo *termui.Par
var terminal *termui.Par
var general *termui.Par
var scoring *termui.Par
var contact *termui.Par

// These are the questions and answers used for the demo.
var q1 = interact.Question{
	Qid:      "1",
	Question: `What is the capital of France?`,
	Options: []*interact.Answer{
		{"1", "Berlin"},
		{"2", "Paris"},
		{"3", "Rome"},
		{"4", "London"},
	},
	Positive: 5.0,
	Negative: 2.5,
}

var q2 = interact.Question{
	Qid:      "1",
	Question: `Which among the following were originally developed at Google?`,
	Options: []*interact.Answer{
		{"1", "Go programming language"},
		{"2", "Ruby"},
		{"3", "Angular"},
		{"4", "Rust"},
	},
	IsMultiple: true,
	Positive:   5.0,
	Negative:   2.5,
}

var q3 = interact.Question{
	Qid:      "1",
	Question: `Which one is the largest ocean in the world?`,
	Options: []*interact.Answer{
		{"1", "Indian"},
		{"2", "Pacific"},
		{"3", "Atlantic"},
		{"4", "Arctic"},
	},
	Positive: 5.0,
	Negative: 2.5,
}

// Question number for the demo.
// TODO(pawan) - Remove this hack.
var currentQn = 1

type State int

const (
	// Denotes if user is seeing the options.
	options State = iota
	// User is being asked to confirm an answer.
	confirmAnswer
	// User is being asked to confirm if they want to skip answering the
	// question.
	confirmSkip
)

// To mantain the state of user while he is answering a question.
var status State

var startTime time.Time
var demoTaken = false

func setupInstructionsPage(th, tw int) {
	instructions = termui.NewPar("")
	instructions.BorderLabel = "Instructions"
	instructions.Height = 50
	instructions.Width = tw
	instructions.PaddingTop = 2

	terminal = termui.NewPar(`
		- Please ensure that you can see all the 4 borders of the Instructions box.
		- If you can't see them, you need to increase the size of your terminal or adjust the font-size to a smaller value.
		- DO NOT proceed with the test, until you are able to see all 4 outer borders of the Instructions box.`)
	terminal.BorderLabel = "Terminal"
	terminal.Height = 8
	terminal.Width = tw
	terminal.PaddingLeft = 2

	// TODO - Take duration from constant.
	general = termui.NewPar(`
		- By taking this test, you agree not to discuss/post the questions shown here.
		- The duration of the test is 60 mins. Timing would be clearly shown.
		- Once you start the test, the timer would not stop, irrespective of any client side issues.
		- Questions can have single or multiple correct answers. They will be shown accordingly.
		- Your total score and the time left at any point in the test would be displayed on the top.
		- You would be given the option to have a second attempt at a question if your first answer is wrong.
		- The scoring for each attempt of a question, would be visible to you in a separate section.
		- At point you can press Ctrl-q to end the test.`)
	general.BorderLabel = "General"
	general.Height = 15
	general.Width = tw
	general.PaddingLeft = 2

	scoring = termui.NewPar(`
		- There is NEGATIVE scoring for wrong answers. So, please DO NOT GUESS.
		- If you skip a question, the score awarded is always ZERO.
		- You might be given the option to recover from negative score with a second attempt.
		- In the above case, please note that another wrong answer would have further negative score.
		- Scoring would be clearly marked in the question on the right hand side box.`)
	scoring.BorderLabel = "Scoring"
	scoring.Height = 10
	scoring.Width = tw
	scoring.PaddingLeft = 2

	contact = termui.NewPar(`
		- If there are any problems with the setup, or something is unclear, please DO NOT start the test.
		- Send email to contact@dgraph.io and tell us the problem. So we can solve it before you take the test.`)
	contact.BorderLabel = "Contact"
	contact.Height = 10
	contact.Width = tw
	contact.PaddingLeft = 2

	demo = termui.NewPar("We have a demo of the how the test would look like. Press s to start the demo.")
	demo.Border = false
	demo.Height = 3
	demo.Width = tw
	demo.TextFgColor = termui.ColorCyan
	demo.PaddingLeft = 2
	demo.PaddingTop = 1
}

func setupQuestionsPage() {
	timeLeft = termui.NewPar(fmt.Sprintf("%dm", testDur))
	timeLeft.Height = 3
	timeLeft.BorderLabel = "Time Left"

	ts := 0.0
	score = termui.NewPar(fmt.Sprintf("%.1f", ts))
	score.BorderLabel = "Total Score"
	score.Height = 3

	que = termui.NewPar("")
	que.BorderLabel = "Question"
	que.PaddingLeft = 1
	que.PaddingRight = 1
	que.PaddingBottom = 1
	que.Height = 33

	s = termui.NewPar("")
	s.BorderLabel = "Scoring"
	s.PaddingTop = 1
	s.PaddingLeft = 1
	s.Height = 33

	a = termui.NewPar("")
	a.TextFgColor = termui.ColorCyan
	a.BorderLabel = "Answers"
	a.PaddingLeft = 1
	a.PaddingRight = 1
	a.PaddingBottom = 1
	a.Height = 14
}

func initializeTest() {
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := interact.NewGruQuizClient(conn)

	// TODO(pawan) - Rename SendQuestion to GetQuestion
	// TODO(pawan) - Have an authenticate method before GetQuestion() to get
	// authenticate the token and get a session token.
	q, err := c.SendQuestion(context.Background(),
		&interact.Req{Repeat: false, Ssid: "testssid", Token: *token})
	if err != nil {
		log.Fatalf("Could not get question.Got err: %v", err)
	}

	setupQuestionsPage()
	renderQuestionsPage()
	populateQuestionsPage(q)
}

func renderInstructionsPage() {
	termui.Render(instructions)
	// Adding an offset so that all these boxes come inside the instructions box.
	termui.Body.Y = 2
	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(10, 1, terminal)),
		termui.NewRow(
			termui.NewCol(10, 1, general)),
		termui.NewRow(
			termui.NewCol(10, 1, scoring)),
		termui.NewRow(
			termui.NewCol(10, 1, contact)),
		termui.NewRow(
			termui.NewCol(10, 1, demo)))

	if demoTaken {
		demo.Text = "Press s to start the test."
	}
	termui.Body.Align()
	termui.Render(termui.Body)

	termui.Handle("/sys/kbd/s", func(e termui.Event) {
		// To clear the instructions box that was rendered.
		termui.Clear()
		// To clear elements of the body.
		termui.Body.Rows = termui.Body.Rows[:0]
		if !demoTaken {
			initializeDemo()
			return
		}
		initializeTest()

	})
}

func renderQuestionsPage() {
	termui.Body.Y = 0
	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(6, 0, timeLeft),
			termui.NewCol(6, 0, score)),
		termui.NewRow(
			termui.NewCol(10, 0, que),
			termui.NewCol(2, 0, s)),
		termui.NewRow(
			termui.NewCol(12, 0, a)))

	termui.Body.Align()
	termui.Render(termui.Body)

	startTime = time.Now()
	termui.Handle("/timer/1s", func(e termui.Event) {
		elapsed := time.Since(startTime)
		left := testDur*time.Minute - elapsed
		// Changing precision to seconds
		r := left % time.Second
		left = left - r

		// Changing the display only every 10 seconds.
		if (left/time.Second)%10 == 0 {
			timeLeft.Text = fmt.Sprintf("%v", left)
			termui.Render(termui.Body)
		}
	})
}

func renderSelectedAnswers(selected []string, m map[string]*interact.Answer) {
	check := "Selected:\n\n"
	for _, k := range selected {
		check += m[string(k)].Ans + "\n"
	}
	check += "\nPress ENTER to confirm. Press any other key to cancel."
	a.Text = check
	status = confirmAnswer
	termui.Render(termui.Body)
}

func optionHandler(e termui.Event, q *interact.Question, selected []string,
	m map[string]*interact.Answer, ansBody string) []string {
	k := e.Data.(termui.EvtKbd).KeyStr

	// For single correct answer qn we just render
	// the selected answer.
	if !q.IsMultiple {
		if status != options {
			return []string{}
		}
		// We append the selected answer and render it.
		selected = append(selected, k)
		renderSelectedAnswers(selected, m)
		return selected
	}
	// For multiple choice questions we check
	// if the user has already selected the answer before.
	exists := false
	for _, key := range selected {
		if key == k {
			exists = true
		}
	}
	// If he hasn't selected the answer before, we display
	// it below the options now.
	if !exists && status == options {
		selected = append(selected, k)
		sort.StringSlice(selected).Sort()
		a.Text = ansBody + strings.Join(selected, ", ")
		a.Text += "\nPress Enter to see chosen options."
		termui.Render(termui.Body)
	}
	return selected
}

func enterHandler(e termui.Event, q *interact.Question, selected []string,
	m map[string]*interact.Answer) {
	// If the user presses enter after selecting options for a
	// multiple choice question.
	if q.IsMultiple && len(selected) > 0 && status == options {
		renderSelectedAnswers(selected, m)
	} else if status == confirmAnswer || status == confirmSkip {
		if currentQn == 0 {
			populateQuestionsPage(&q1)
		}
		if currentQn == 1 {
			populateQuestionsPage(&q2)
		} else if currentQn == 2 {
			populateQuestionsPage(&q3)
		} else if currentQn == 3 {
			termui.Clear()
			termui.Body.Rows = termui.Body.Rows[:0]
			demoTaken = true
			renderInstructionsPage()
			currentQn = -1
		}
		currentQn += 1
	}
}

func keyHandler(ansBody string, selected []string) []string {
	a.Text = ansBody
	selected = selected[:0]
	status = options
	termui.Render(termui.Body)
	return selected
}

func populateQuestionsPage(q *interact.Question) {
	que.Text = q.Question
	s.Text = fmt.Sprintf("Right answer => +%1.1f\n\nWrong answer => -%1.1f",
		q.Positive, q.Negative)

	// Selected contains the options user has already selected.
	selected := []string{}
	// This is the body of the answer which has all the options.
	ansBody := ""
	// Map m contains a map of the key to select an answer and the answer
	// corresponding to it.
	m := make(map[string]*interact.Answer)
	var buf bytes.Buffer

	status = options
	if q.IsMultiple {
		buf.WriteString("This question could have multiple correct answers.\n\n")
	} else {
		buf.WriteString("This question only has a single correct answer.\n\n")
	}
	opt := 'a'
	for _, o := range q.Options {
		buf.WriteRune(opt)
		buf.WriteRune(')')
		buf.WriteRune(' ')
		buf.WriteString(o.Ans)
		buf.WriteRune('\n')
		m[string(opt)] = o
		opt++
	}
	buf.WriteString("\ns) Skip question\n\n")
	// We store this so that this can be rendered later based on different
	// key press.
	ansBody = buf.String()
	a.Text = ansBody
	termui.Render(termui.Body)

	// Attaching event handlers on the options for a answer.
	for i := 'a'; i < opt; i++ {
		termui.Handle(fmt.Sprintf("/sys/kbd/%c", i),
			func(e termui.Event) {
				selected = optionHandler(e, q, selected, m,
					ansBody)
			})
	}

	termui.Handle("/sys/kbd/s", func(e termui.Event) {
		a.Text = "Are you sure you want to skip the question? \n\nPress ENTER to confirm. Press any other key to cancel."
		termui.Render(termui.Body)
		status = confirmSkip
	})

	termui.Handle("/sys/kbd/<enter>", func(e termui.Event) {
		enterHandler(e, q, selected, m)
	})

	// On any other keypress we reset the answer text and the selected answers.
	termui.Handle("/sys/kbd", func(e termui.Event) {
		selected = keyHandler(ansBody, selected)
	})
}

func initializeDemo() {
	setupQuestionsPage()
	renderQuestionsPage()
	populateQuestionsPage(&q1)
}

func main() {
	flag.Parse()
	err := termui.Init()
	if err != nil {
		panic(err)
	}
	defer termui.Close()

	th := termui.TermHeight()
	tw := termui.TermWidth()

	setupInstructionsPage(th, tw)
	renderInstructionsPage()

	// Pressing Ctrl-q terminates the ui.
	termui.Handle("/sys/kbd/C-q", func(termui.Event) {
		termui.StopLoop()
	})
	termui.Loop()
}