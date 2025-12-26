package main

// Future scope
// Confirmation prompt before deleting, look up tview dialog widget
// Preview pane on the right, read below comment
// For every j/k event, update the selected item global variable - if folder, do ls on that folder and display in preview pane
// if file, do cat on that file and display in preview pane

import (
	"fmt"
	"strings"
	"log"
	"os"
	"os/exec"
	"syscall"
	"github.com/rivo/tview"
	"github.com/gdamore/tcell/v2"
)

var (
	bufferCmd string
	buffer string
	path = "/home/metastag"
)

// Returns true if path is a folder, else return false
func checkFolder(input string) bool {
	info, err := os.Stat(path + "/" +  input)
	if err != nil {
		log.Println("Error in checkFolder() - ", err)
		return false
	}
	if info.IsDir() {
		return true
	}
	return false
}

// Returns parent folder of path
func parentFolder(path string) string {
	index := strings.LastIndex(path, "/")
	if index == 0 {
		return "/"
	}
	return path[:index]
}

// Update path to a child folder
func navigateForward(child string) {
	path = path + "/" + child
}

// Update path to it's parent folder
func navigateBackward() {
	path = parentFolder(path)
}

// Helper function to refresh all panes together
func refreshPanes(left *tview.List, center *tview.List) {
	refreshPane(left, parentFolder(path))
	refreshPane(center, path)
}

// Write the contents of a pane from folder given
func refreshPane(pane *tview.List, folder string) {
	pane.Clear() // empty pane

	// Read contents of folder, return any error
	files, err := os.ReadDir(folder)
	if err != nil {
		log.Fatal("Error in refreshPane() - ", err)
	}

	// Write the contents to pane line by line
	for _, file := range files {
		if file.IsDir() { // If folder, mark as red
			name := "[red]" + file.Name() + "[white]"
			pane.AddItem(name, "", 0, nil)
		} else {
			pane.AddItem(file.Name(), "", 0, nil)
		}
	}
}

// Open folder/file - called by pressing l or Enter
func openEvent(left *tview.List, center *tview.List) {
	// If empty folder, do nothing
	if center.GetItemCount() == 0 {
		return
	}
	// Get selected item
	child, _ := center.GetItemText(center.GetCurrentItem())

	// get index of [white] in string to differentiate file and folder
	end := strings.LastIndex(child, "[white]")

	// if there is no [white], it is a file, open with xdg-open
	if end == -1 {
		cmd := exec.Command("xdg-open", path + "/" + child)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setsid: true,
		}
		cmd.Start()
		return
	}

	child = child[5:end] // remove [red] and [white] from start and end of string
	navigateForward(child) // update path to new folder
	refreshPanes(left, center) // update panes
}

// Delete selected file, ignore if folder for safety reasons
func deleteEvent(title *tview.TextView, left *tview.List, center *tview.List) {
	// Get selected item
	child, _ := center.GetItemText(center.GetCurrentItem())

	// get index of [white] in string to differentiate file and folder
	end := strings.LastIndex(child, "[white]")

	// if there is [white], it is a folder, ignore
	if end != -1 {
		return
	}

	// Delete file
	cmd := exec.Command("rm", path + "/" + child)
	_, err := cmd.Output()
	if err != nil {
		log.Println("Error in deleteEvent() - ", err)
	}
	// Refresh panes
	refreshPanes(left, center)
}

// Copy selected file, saves to buffer until pasteEvent() is called
func copyEvent(center *tview.List) {
	// Get selected item
	child, _ := center.GetItemText(center.GetCurrentItem())

	// get index of [white] in string to remove it
	end := strings.LastIndex(child, "[white]")
	if end != -1 {
		child = child[5:end] // remove [red] and [white] from start and end of string
	}

	// Save path to buffer, to be used when pasteEvent() is called
	bufferCmd = "cp"
	buffer = path + "/" + child
}

// Cut selected file, saves to buffer until pasteEvent() is called
func cutEvent(center *tview.List) {
	// Get selected item
	child, _ := center.GetItemText(center.GetCurrentItem())

	// get index of [white] in string to remove it
	end := strings.LastIndex(child, "[white]")
	if end != -1 {
		child = child[5:end] // remove [red] and [white] from start and end of string
	}

	// Save path to buffer, to be used when pasteEvent() is called
	bufferCmd = "mv"
	buffer = path + "/" + child

}

// Pastes file in buffer to current folder
func pasteEvent(left *tview.List, center *tview.List) {
	cmd := exec.Command(bufferCmd, buffer, path)
	_, err := cmd.Output()
	if err != nil {
		log.Println("Error in pasteEvent() - ", err)
	}
	refreshPanes(left, center) // refresh panes
}

func main() {
	// Initialize tview application
	app := tview.NewApplication()
	
	// Title widget
	title := tview.NewTextView() 
	fmt.Fprintln(title, path)

	// Left pane widget
	left := tview.NewList().
		ShowSecondaryText(false)

	left.SetBorder(true).SetTitle("Previous Folder")

	// Center pane widget
	center := tview.NewList().
		ShowSecondaryText(false)

	center.SetBorder(true).SetTitle("Current Folder")

	// Define keybindings
	center.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'h' { // h - go back one folder
			navigateBackward()
			refreshPanes(left, center)
		} else if event.Rune() == 'l' { // l - Open folder/file
			openEvent(left, center)
		} else if event.Rune() ==  'j' { // j - go down one cell
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	    	} else if event.Rune() == 'k' { // k - go up one cell
			return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		} else if event.Rune() == 'G' { // G - navigate to bottom
			return tcell.NewEventKey(tcell.KeyPgDn, 0, tcell.ModNone)
		} else if event.Rune() == 'g' { // g - navigate to top
			return tcell.NewEventKey(tcell.KeyPgUp, 0, tcell.ModNone)
		} else if event.Rune() == 13 { // Enter - Open folder/file
			openEvent(left, center)
		} else if event.Rune() == 'd' { // d - delete selected file
			deleteEvent(title, left, center)
		} else if event.Rune() == 'c' { // c - copy selected file
			copyEvent(center)
		} else if event.Rune() == 'x' { // x - cut selected file
			cutEvent(center)
		} else if event.Rune() == 'p' { // p - paste in current folder
			pasteEvent(left, center)
		}
		return event
	})

	// Grid widget
	grid := tview.NewGrid().
		SetRows(2,0).
		SetColumns(50,0).
		AddItem(title, 0, 0, 1, 3, 0, 0, false)
		
	// Layout for screens narrower than 100 cells (left is hidden)
	grid.AddItem(left, 0, 0, 0, 0, 0, 0, false).
		AddItem(center, 1, 0, 1, 3, 0, 0, false)

	// Layout for screens wider than 100 cells.
	grid.AddItem(left, 1, 0, 1, 1, 0, 100, false).
		AddItem(center, 1, 1, 1, 2, 0, 100, false)

	refreshPanes(left, center)


	// Run app
	if err := app.SetRoot(grid, true).SetFocus(center).Run(); err != nil {
		panic(err)
	}
}
