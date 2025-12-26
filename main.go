package main

import (
	"fmt"
	"strings"
	"os"
	"os/exec"
	"syscall"
	"github.com/rivo/tview"
	"github.com/gdamore/tcell/v2"
)

var (
	selected string // currently selected item
	selectedType string // file or folder
	bufferCmd string // whether copy or cut operation
	buffer string // store path of item to copy/cut
	path = "/home/metastag" // starting location
)

// Returns true if path is a folder, else return false
func checkFolder(input string) bool {
	info, err := os.Stat(path + "/" +  input)
	if err != nil {
		fmt.Fprintf(os.Stdout, "Error in checkFolder() - ", err)
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
func refreshPanes(left *tview.List, center *tview.List, preview *tview.TextView) {
	refreshPane(left, parentFolder(path))
	refreshPane(center, path)
}

// Write the contents of a pane from folder given
func refreshPane(pane *tview.List, folder string) {
	pane.Clear() // empty pane

	// Read contents of folder, return any error
	files, err := os.ReadDir(folder)
	if err != nil {
		fmt.Fprintf(os.Stdout, "Error in refreshPane() - ", err)
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

// Update preview pane
func refreshPreview(pane *tview.TextView) {
	var cmd *exec.Cmd
	if selectedType == "file" && strings.Contains(selected, ".txt") {
		cmd = exec.Command("cat", path + "/" + selected)
	} else if selectedType == "folder" {
		cmd = exec.Command("ls", path + "/" + selected)
	} else {
		return
	}

	out, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stdout, "Error in refreshPreview() - ", err)
	}
	pane.Clear()
	fmt.Fprintln(pane, string(out))
}

// Open folder/file - called by pressing l or Enter
func openEvent(left *tview.List, center *tview.List, preview *tview.TextView) {
	// If empty folder, do nothing
	if center.GetItemCount() == 0 {
		return
	}

	// if selected item is file, open with xdg-open
	if selectedType == "file" {
		cmd := exec.Command("xdg-open", path + "/" + selected)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setsid: true,
		}
		cmd.Start()
		return
	}

	navigateForward(selected) // update path to new folder
	refreshPanes(left, center, preview) // update panes
}

// Delete selected file, ignore if folder for safety reasons
func deleteEvent(app *tview.Application, pages *tview.Pages, left *tview.List, center *tview.List, preview *tview.TextView) {
	// If folder, ignore
	if selectedType == "folder" {
		return
	}

	// Confirmation dialog box
	modal := tview.NewModal().
		SetText("Delete " + selected + " file?").
		AddButtons([]string{"Yes", "No, cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Yes" {
				// Delete file
				cmd := exec.Command("rm", path + "/" + selected)
				_, err := cmd.Output()
				if err != nil {
					fmt.Fprintf(os.Stdout, "Error in deleteEvent() - ", err)
				}
				// Refresh panes
				refreshPanes(left, center, preview)
			}
			pages.RemovePage("modal")
			app.SetFocus(center)
		})
	
	pages.AddPage("modal", modal, true, true)
	app.SetFocus(modal)
}

// Copy selected file, saves to buffer until pasteEvent() is called
func copyEvent(center *tview.List) {
	// Save path to buffer, to be used when pasteEvent() is called
	bufferCmd = "cp"
	buffer = path + "/" + selected
}

// Cut selected file, saves to buffer until pasteEvent() is called
func cutEvent(center *tview.List) {
	// Save path to buffer, to be used when pasteEvent() is called
	bufferCmd = "mv"
	buffer = path + "/" + selected
}

// Pastes file in buffer to current folder
func pasteEvent(left *tview.List, center *tview.List, preview *tview.TextView) {
	cmd := exec.Command(bufferCmd, buffer, path)
	_, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stdout, "Error in pasteEvent() - ", err)
	}
	refreshPanes(left, center, preview) // refresh panes
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

	// Preview pane widget
	preview := tview.NewTextView()

	preview.SetBorder(true).SetTitle("Preview Pane")

	// Grid widget
	grid := tview.NewGrid().
		SetRows(2,0).
		SetColumns(30,0,30).
		AddItem(title, 0, 0, 1, 3, 0, 0, false)
		
	// Layout for screens narrower than 100 cells (left is hidden)
	grid.AddItem(left, 0, 0, 0, 0, 0, 0, false).
		AddItem(center, 1, 0, 1, 3, 0, 0, false)

	// Layout for screens wider than 100 cells.
	grid.AddItem(left, 1, 0, 1, 1, 0, 100, false).
		AddItem(center, 1, 1, 1, 1, 0, 100, false).
		AddItem(preview, 1, 2, 1, 1, 0, 100, false)

	refreshPanes(left, center, preview)


	// Wrap the grid in a pages widget (required to display modal during deletion)
	pages := tview.NewPages()
	pages.AddPage("main", grid, true, true)

	// Define keybindings
	center.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'h' { // h - go back one folder
			navigateBackward()
			refreshPanes(left, center, preview)
		} else if event.Rune() == 'l' { // l - Open folder/file
			openEvent(left, center, preview)
		} else if event.Rune() ==  'j' { // j - go down one cell
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	    	} else if event.Rune() == 'k' { // k - go up one cell
			return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		} else if event.Rune() == 'G' { // G - navigate to bottom
			return tcell.NewEventKey(tcell.KeyPgDn, 0, tcell.ModNone)
		} else if event.Rune() == 'g' { // g - navigate to top
			return tcell.NewEventKey(tcell.KeyPgUp, 0, tcell.ModNone)
		} else if event.Rune() == 13 { // Enter - Open folder/file
			openEvent(left, center, preview)
		} else if event.Rune() == 'd' { // d - delete selected file
			deleteEvent(app, pages, left, center, preview)
		} else if event.Rune() == 'c' { // c - copy selected file
			copyEvent(center)
		} else if event.Rune() == 'x' { // x - cut selected file
			cutEvent(center)
		} else if event.Rune() == 'p' { // p - paste in current folder
			pasteEvent(left, center, preview)
		} else if event.Rune() == 'q' { // q - quit
			app.Stop()
		}
		return event
	})

	center.SetChangedFunc(func(index int, main, secondary string, shortcut rune) {
		// Get index of [white] in string to differentiate file and folder
		end := strings.LastIndex(main, "[white]")

		// If there is no [white], it is a file
		if end == -1 {
			selected = main
			selectedType = "file"
			refreshPreview(preview)
		} else {
			main = main[5:end] // remove [red] and [white] from start and end of string
			selected = main
			selectedType = "folder"
			refreshPreview(preview)
		}
	})

	// Run app
	if err := app.SetRoot(pages, true).SetFocus(center).Run(); err != nil {
		panic(err)
	}
}
