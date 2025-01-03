package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jaredmontoya/lingo/src/audioPlayer"
	"github.com/jaredmontoya/lingo/src/fileReader"
	"github.com/jaredmontoya/lingo/src/interfaceLanguage"
	"github.com/jaredmontoya/lingo/src/languageHandler"
	"github.com/jaredmontoya/lingo/src/strokeOrder"
	"github.com/jaredmontoya/lingo/src/terminalSize"
	"github.com/jaredmontoya/lingo/src/translator"
)

//go:embed src/translator/hanzi.json
var hanzi []byte

// Styles for the app
var (
	// This is the style for the titles in the menus
	titleStyle = lipgloss.NewStyle().MarginLeft(2)
	// Style for the items to select in the menus
	itemStyle = lipgloss.NewStyle().PaddingLeft(4)
	// Style for the currently selected item in the menu
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	// This is the style for the "quit text", i.e the text that tells us how to quit the program.
	quitTextStyle = lipgloss.NewStyle().Margin(1, 0, 2, 4)
	// Styles for the different levels of word knowledge
	notKnownItemStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")) // 1) not known --> red
	semiKnownItemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFCA3A")) // 2) semiknown --> yellow
	knownItemStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#00b300")) // 3) known --> green
)

// Styles for the tables used in the menus
var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

/*
visitFile function:
this function appends the list of files inside a given directory inside the filePaths slice.
*/
func visitFile(fp string, fi os.DirEntry, err error) error {
	if err != nil {
		fmt.Println(err) // can't walk here,
		return nil       // but continue walking elsewhere
	}
	if fi.IsDir() {
		return nil // not a file. ignore.
	}
	// Append the file path to the slice
	filePaths = append(filePaths, fp)
	return nil
}

/*
listDirectories function:
this function lists the subdirectories inside a certain path. it returns the list in the
form of a slice and a possible error (which is nil if everything went as expected)
*/
func listDirectories(directoryPath string) ([]string, error) {
	// declare the slice variable we're going to return
	var directories []string

	// here we're using the filepath package we imported at the beginning
	err := filepath.Walk(directoryPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// if the files we're encountering while "walking" are a directories, append them to the directories slice.
			directories = append(directories, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// return the slice and the error (which if we reached this point will be nil since no error occured)
	return directories, nil
}

var filePaths []string // Declare a global slice to store file paths

type model struct {
	choices         []string        // text-file selection menu (OLD)
	choices2        []string        // language select menu (OLD)
	cursor          int             // which to-do list item our cursor is pointing at
	viewIndex       int             // viewIndex --> will be 0 for the menu, and 1 for an opened file.
	openedFile      string          // will store the name of the file we opened.
	openedFileText  fileReader.Text // will store the fileReader.Text object representing the file we opened
	cursor2         int             //
	currentLanguage string          // This is the current language we're studying
	currentError    string          // This is the current error that go cathced; in this way, if something goes wrong with the APIs the app
	// doesn't crash, it just notifies the user and tells you what went wrong.
	interfaceLanguage string              // This is the language of the interface (if you're reading this I assume you speak english and thus it will be english)
	languageTable     table.Model         // This is the table listing all the languages (new UI)
	textTable         table.Model         // This is the table listing all the text files we can open inside the app (new UI).
	hanziData         map[string][]string // This is the map that stores the pinyin equivalent of the most common hanzi in simplified mandarin chinese
}

/*
This function initializes the bubbletea model to boot the application; this is one of the "dirtiest" parts of the application
in terms of code and it needs a serious refactor.
*/
func initialModel() model {
	// Get the interface language from the interfaceLanguage.txt file by reading it.
	interfaceLang, _ := os.ReadFile("setup/interfaceLanguage.txt")
	// Convert the []byte object we got into a string
	interfaceLangString := string(interfaceLang)
	// These 4 lines of code just make sure to remove all the possible spaces, new lines and tabular spaces.
	stripped := strings.ReplaceAll(interfaceLangString, "\t", "")
	stripped = strings.ReplaceAll(stripped, " ", "")
	stripped = strings.ReplaceAll(stripped, "\n", "")
	interfaceLangString = stripped

	languageTitle := interfaceLanguage.InterfaceLanguage[interfaceLanguage.LanguagesCodeMap[interfaceLangString]][0]
	columns := []table.Column{
		{Title: languageTitle, Width: 35},
	}
	directoryPath := "languages"

	// Store the subdirectories of the "languages" directory inside the directories slice;
	// this lists all the languages we can study with this app.
	directories, _ := listDirectories(directoryPath)
	// skip the first one tho because it will be the directory "languages" itself;
	// (this is because of how the filepath package works)
	directories = directories[1:]

	// All this code just creates the tables for the UI; I will place this into another file soon and refactor it.
	var rows []table.Row

	for _, v := range directories {
		e := table.Row{v[10:]}
		rows = append(rows, e)
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)
	textTitle := interfaceLanguage.InterfaceLanguage[interfaceLanguage.LanguagesCodeMap[interfaceLangString]][3]
	columns2 := []table.Column{
		{Title: textTitle, Width: 35},
	}

	var rows2 []table.Row

	for _, v := range filePaths {
		e := table.Row{v[6:]}
		rows2 = append(rows2, e)
	}

	t2 := table.New(
		table.WithColumns(columns2),
		table.WithRows(rows2),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	t2.SetStyles(s)

	// return the model object we need to start the bubbletea app.
	return model{
		choices:           filePaths,
		choices2:          directories,
		viewIndex:         2,
		cursor2:           0,
		currentError:      "",
		interfaceLanguage: interfaceLangString,
		languageTable:     t,
		textTable:         t2,
		hanziData:         translator.InitHanzi(hanzi),
	}
}

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

/*
Update method; this update method handles the changes that need to be done to the UI
when a key is pressed or when something is changed (in according with the ELM architecture principles).
*/
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.openedFileText.PageList = fileReader.DivideInPages(m.openedFileText.TokenList)
	m.openedFileText.Pages = len(m.openedFileText.PageList)
	if m.openedFileText.CurrentPage > m.openedFileText.Pages-1 {
		m.openedFileText.CurrentPage = 0
	}
	m.languageTable, _ = m.languageTable.Update(msg)
	m.textTable, _ = m.textTable.Update(msg)
	switch m.viewIndex {
	case 0:
		switch msg := msg.(type) {

		// Is it a key press?
		case tea.KeyMsg:

			// Cool, what was the actual key pressed?
			switch msg.String() {

			// These keys should exit the program.
			case "ctrl+c", "q":
				return m, tea.Quit

			// The "up" and "k" keys move the cursor up
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}

			// The "down" and "j" keys move the cursor down
			case "down", "j":
				if m.cursor < len(m.choices)-1 {
					m.cursor++
				}
			case "b":
				m.viewIndex = 2
				m.currentLanguage = ""

			// The "enter" key and the spacebar (a literal space) toggle
			// the selected state for the item that the cursor is pointing at.
			case "enter", " ":
				m.viewIndex = 1
				m.openedFile = "texts/" + m.textTable.SelectedRow()[0]
				text := fileReader.InitText(m.openedFile, m.currentLanguage)
				m.openedFileText = text
			// If the key pressed is f, generate a dictionary file.
			case "f":
				dictionary := fileReader.MakeDictFromMenu(m.currentLanguage)
				fileReader.MakeDictionary(dictionary, m.currentLanguage, m.interfaceLanguage)
			case "z":
				dictionary := fileReader.MakeDictFromMenu(m.currentLanguage)
				fileReader.MakeAltDictionary(dictionary, m.currentLanguage, m.interfaceLanguage, m.hanziData)
			}
		}

		// Return the updated model to the Bubble Tea runtime for processing.
		// Note that we're not returning a command.
	case 1:
		switch msg := msg.(type) {

		// Is it a key press?
		case tea.KeyMsg:

			// Cool, what was the actual key pressed?
			switch msg.String() {

			// These keys should exit the program.
			case "ctrl+c", "q":
				return m, tea.Quit

			// The "up" and "k" keys move the cursor up
			case "left", "h":
				if m.openedFileText.TokenCursorPosition > 0 {
					m.openedFileText.TokenCursorPosition--
				}

			// The "down" and "j" keys move the cursor down
			case "right", "l":
				if m.openedFileText.TokenCursorPosition < m.openedFileText.TokenLength-1 {
					m.openedFileText.TokenCursorPosition++
				}

			case "up", "k":
				line := terminalSize.GetWordsPerLine()
				if m.openedFileText.TokenCursorPosition > 0 && m.openedFileText.TokenCursorPosition-line >= 0 {
					m.openedFileText.TokenCursorPosition -= line
				}
			case "down", "j":
				line := terminalSize.GetWordsPerLine()
				if m.openedFileText.TokenCursorPosition < m.openedFileText.TokenLength-1 && m.openedFileText.TokenCursorPosition+line < m.openedFileText.TokenLength-1 {
					m.openedFileText.TokenCursorPosition += line
				}

			// We use a and d to move between pages
			// If the key pressed is d, move one page to the right
			case "d":
				/*
					Check that we're not on the last page first; if we were on the last page
					we would ideally not want to be able to go one page further since there's no
					next page by definition
					fmt.Println(m.openedFileText.CurrentPage, m.openedFileText.Pages)
				*/
				if m.openedFileText.CurrentPage < m.openedFileText.Pages-1 {
					// If everything is alright implement the logic, i.e augment the page counter and go to the next page
					m.openedFileText.CurrentPage++
				}
			// If the key pressed is a, move one page to the left
			case "a":
				// Check if we're not on page 0 first: if we were, we would go to page -1 and that doesn't make sense.
				if m.openedFileText.CurrentPage > 0 {
					// If everything is alright implement the logic, i.e decrement the page counter and go to the next page
					m.openedFileText.CurrentPage--
				}

			case "0":
				m.openedFileText.WordLevels[m.openedFileText.TokenList[m.openedFileText.TokenCursorPosition]] = 0
				fileReader.MakeJsonFile(m.openedFileText.WordLevels, m.currentLanguage)
			case "1":
				m.openedFileText.WordLevels[m.openedFileText.TokenList[m.openedFileText.TokenCursorPosition]] = 1
				fileReader.MakeJsonFile(m.openedFileText.WordLevels, m.currentLanguage)
			case "2":
				m.openedFileText.WordLevels[m.openedFileText.TokenList[m.openedFileText.TokenCursorPosition]] = 2
				fileReader.MakeJsonFile(m.openedFileText.WordLevels, m.currentLanguage)
			case "3":
				m.openedFileText.WordLevels[m.openedFileText.TokenList[m.openedFileText.TokenCursorPosition]] = 3
				fileReader.MakeJsonFile(m.openedFileText.WordLevels, m.currentLanguage)

			case "4":
				currentLanguageId := languageHandler.LanguageMap[m.currentLanguage]
				m.currentError = ""
				m.currentError += audioPlayer.GetAudio(m.openedFileText.TokenList[m.openedFileText.TokenCursorPosition], currentLanguageId)
				mp3FilePath := fmt.Sprintf("audio/%s.mp3", m.openedFileText.TokenList[m.openedFileText.TokenCursorPosition])

				if m.currentError != "" {
					m.currentError += "\n"
				}
				m.currentError += audioPlayer.PlayMP3(mp3FilePath)
				m.currentError += "\n"
				m.currentError += audioPlayer.DeleteMP3(mp3FilePath)

			// get translation
			case "5":
				currentlLanguageId := languageHandler.LanguageMap2[m.currentLanguage]
				translation, errString := translator.Translate2(m.openedFileText.TokenList[m.openedFileText.TokenCursorPosition], currentlLanguageId, m.interfaceLanguage)
				m.currentError = errString
				m.openedFileText.CurrentTranslate = translation

			case "6":
				m.openedFileText.CurrentLatinization = translator.LatinizeText(m.openedFileText.TokenList[m.openedFileText.TokenCursorPosition], m.hanziData, m.currentLanguage)
			case "7":
				url := fmt.Sprintf("https://www.strokeorder.com/chinese/%s", m.openedFileText.TokenList[m.openedFileText.TokenCursorPosition])
				err := strokeOrder.OpenBrowser(url)
				if err != nil {
					m.currentError += err.Error()
				}
			case "8":
				url := fmt.Sprintf("https://translate.google.com/?sl=%s&tl=%s&text=%s&op=translate", languageHandler.LanguageMap2[m.currentLanguage], m.interfaceLanguage, m.openedFileText.TokenList[m.openedFileText.TokenCursorPosition])
				err := strokeOrder.OpenBrowser(url)
				if err != nil {
					m.currentError += err.Error()
				}
			case "9":
				currentlLanguageId := languageHandler.LanguageMap2[m.currentLanguage]
				translation, errString := translator.Translate(m.openedFileText.TokenList[m.openedFileText.TokenCursorPosition], currentlLanguageId, m.interfaceLanguage)
				m.currentError = errString
				m.openedFileText.CurrentTranslate = translation

			case "f":
				fileReader.MakeDictionary(m.openedFileText.WordLevels, m.currentLanguage, m.interfaceLanguage)
			case "z":
				fileReader.MakeAltDictionary(m.openedFileText.WordLevels, m.currentLanguage, m.interfaceLanguage, m.hanziData)

			// Move the cursor to the beginning of the current page.
			case "m":
				currentCursor := m.openedFileText.CurrentPage * terminalSize.GetLinesPerPage() * terminalSize.GetWordsPerLine()
				m.openedFileText.TokenCursorPosition = currentCursor

			// The "enter" key and the spacebar (a literal space) toggle
			// the selected state for the item that the cursor is pointing at.
			case "b":
				m.viewIndex = 0
				m.currentError = ""
			}
		}
	case 2:
		switch msg := msg.(type) {

		// Is it a key press?
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "up", "k":
				if m.cursor2 > 0 {
					m.cursor2--
				}

			// The "down" and "j" keys move the cursor2 down
			case "down", "j":
				if m.cursor2 < len(m.choices2)-1 {
					m.cursor2++
				}

			// The "enter" key and the spacebar (a literal space) toggle
			// the selected state for the item that the cursor is pointing at.
			case "enter", " ":
				m.viewIndex = 0
				m.currentLanguage = m.languageTable.SelectedRow()[0]

			}

		}

	}
	return m, nil
}

func (m model) View() string {
	var s string
	if m.viewIndex == 0 {
		// The header
		s = interfaceLanguage.InterfaceLanguage[interfaceLanguage.LanguagesCodeMap[m.interfaceLanguage]][2] + m.currentLanguage + "\n"
		s += "\n"

		// Iterate over our choices

		s += baseStyle.Render(m.textTable.View())

		// The footer
		s += interfaceLanguage.InterfaceLanguage[interfaceLanguage.LanguagesCodeMap[m.interfaceLanguage]][4]
		s += interfaceLanguage.InterfaceLanguage[interfaceLanguage.LanguagesCodeMap[m.interfaceLanguage]][5]
		s += "\n" + interfaceLanguage.InterfaceLanguage[interfaceLanguage.LanguagesCodeMap[m.interfaceLanguage]][1] + "\n"
	} else if m.viewIndex == 1 {
		wordsPerLine := terminalSize.GetWordsPerLine()
		linesPerPage := terminalSize.GetLinesPerPage()
		width, height := terminalSize.GetTerminalSize()

		s = interfaceLanguage.InterfaceLanguage[interfaceLanguage.LanguagesCodeMap[m.interfaceLanguage]][6] + m.openedFile + interfaceLanguage.InterfaceLanguage[interfaceLanguage.LanguagesCodeMap[m.interfaceLanguage]][7]
		s += fmt.Sprintf("%v", m.openedFileText.TokenCursorPosition)
		s += fmt.Sprintf("\n%s %v %v\n", interfaceLanguage.InterfaceLanguage[interfaceLanguage.LanguagesCodeMap[m.interfaceLanguage]][8], width, height)
		s += "\n"
		for k, element := range m.openedFileText.PageList[m.openedFileText.CurrentPage] {
			var padding1 string = ""
			var padding2 string = " "
			if k%wordsPerLine == 0 && k != 0 {
				padding2 = "\n"
			}
			s += padding1
			actualKey := k + (m.openedFileText.CurrentPage * wordsPerLine * linesPerPage)
			if actualKey == m.openedFileText.TokenCursorPosition {
				s += selectedItemStyle.Render(element)
			} else if value, ok := m.openedFileText.WordLevels[element]; ok {
				switch value {
				case 0:
					s += element
				case 1:
					s += notKnownItemStyle.Render(element)
				case 2:
					s += semiKnownItemStyle.Render(element)
				case 3:
					s += knownItemStyle.Render(element)
				default:
					s += element
				}
			} else {
				s += element
			}
			s += padding2
		}
		s += "\n"
		s += fmt.Sprintf("%v", m.openedFileText.CurrentPage)
		s += fmt.Sprintf("\n%s %v", interfaceLanguage.InterfaceLanguage[interfaceLanguage.LanguagesCodeMap[m.interfaceLanguage]][9], m.openedFileText.Pages)
		s += "\n"
		s += fmt.Sprintf("%s %s", interfaceLanguage.InterfaceLanguage[interfaceLanguage.LanguagesCodeMap[m.interfaceLanguage]][10], m.openedFileText.CurrentTranslate)
		s += "\n"
		s += fmt.Sprintf("%s %s", interfaceLanguage.InterfaceLanguage[interfaceLanguage.LanguagesCodeMap[m.interfaceLanguage]][13], m.openedFileText.CurrentLatinization)
		s += "\n" + interfaceLanguage.InterfaceLanguage[interfaceLanguage.LanguagesCodeMap[m.interfaceLanguage]][11] + m.currentError
		s += "\n" + interfaceLanguage.InterfaceLanguage[interfaceLanguage.LanguagesCodeMap[m.interfaceLanguage]][12] + "\n" + interfaceLanguage.InterfaceLanguage[interfaceLanguage.LanguagesCodeMap[m.interfaceLanguage]][1]
	} else if m.viewIndex == 2 {
		return baseStyle.Render(m.languageTable.View()) + "\n"
	}

	// Send the UI for rendering
	return s
}

// This is the main function, the main entrypoint of the program.
func main() {
	// Run the WalkDir function we looked at the beginning of the file inside of the texts directory
	// store the possible error we could get inside of the err variable
	err := filepath.WalkDir("./texts", visitFile)
	// If there is no error, it's all right; print it out to the console.
	if err != nil {
		fmt.Print("All right")
	}
	// Create a new bubbleTea program using the model returned by the initalModel() function
	// We saw what that function does at the beginning of the program.
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	// If there's an error in the running of the application, let the user know
	// by printing it out to the console.
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
