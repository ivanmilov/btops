package handlers

import (
	"btops/config"
	"btops/monitors"
	"fmt"
	"log"
	"strconv"
	"strings"
)

type baseHandler struct {
	config *config.Config
}

type handlerFunc func(*monitors.Monitors) bool

type AppendHandler struct{ baseHandler }
type RemoveHandler struct{ baseHandler }
type RenameHandler struct {
	baseHandler
	renamers []Renamer
}

type Renamer interface {
	Initialize(*config.Config)
	CanRename(*monitors.Desktop, int) bool
	Rename(*monitors.Desktop, int) bool
}

type constantRenamer struct{ name string }
type staticRenamer struct{ names []string }
type clientRenamer struct {
	max_length  int
	unique_name bool
}
type numericRenamer struct{}
type classifiedRenamer struct {
	priorityMap map[string]classification
	unique_name bool
}

type classification struct {
	name     string
	priority int
}

type Handler interface {
	Initialize(*config.Config)
	ShouldHandle() bool
	Handle(*monitors.Monitors) bool
}

type Handlers []Handler

func NewHandlers(c *config.Config) *Handlers {
	h := Handlers{
		&AppendHandler{},
		&RemoveHandler{},
		&RenameHandler{},
	}

	for i := range h {
		h[i].Initialize(c)
	}

	return &h
}

func (h Handlers) Handle(m *monitors.Monitors) {
	for _, handler := range h {
		if !handler.ShouldHandle() {
			continue
		}

		if handler.Handle(m) {
			return
		}
	}
}

func (b *baseHandler) Initialize(c *config.Config) {
	b.config = c
}

func (a AppendHandler) ShouldHandle() bool {
	return a.config.AppendWhenOccupied || a.config.Min > 1
}

func (a AppendHandler) Handle(m *monitors.Monitors) bool {
	for _, monitor := range *m {
		dCount := len(monitor.Desktops)

		if a.config.Max <= dCount {
			continue
		}

		if !a.config.AppendWhenOccupied && dCount >= a.config.Min {
			continue
		}

		appendDesktop := false
		for i, desktop := range monitor.Desktops {
			if dCount < a.config.Min {
				appendDesktop = true
				break
			}

			if desktop.IsEmpty() {
				break
			}

			if i == dCount-1 {
				appendDesktop = true
			}
		}

		if !appendDesktop {
			continue
		}

		err := monitor.AppendDesktop(strconv.Itoa(dCount + 1))
		if err != nil {
			log.Println("Unable to append desktop to monitor: ", monitor.Name, err)
			continue
		}

		// return true
	}

	return false
}

func (r RemoveHandler) ShouldHandle() bool {
	return r.config.RemoveEmpty
}

func (r RemoveHandler) Handle(m *monitors.Monitors) bool {
	for _, monitor := range *m {
		for _, desktop := range monitor.EmptyDesktops() {
			if *desktop == monitor.Desktops[len(monitor.Desktops)-1] {
				continue
			}

			if r.config.Min >= len(monitor.Desktops) {
				continue
			}

			err := monitor.RemoveDesktop(desktop.Id)
			if err != nil {
				log.Println("Unable to remove desktop: ", desktop.Name, err)
				continue
			}

			// return true
		}
	}

	return false
}

func (r *RenameHandler) Initialize(c *config.Config) {
	r.baseHandler.Initialize(c)
	r.renamers = *NewRenamers(c)
}

func NewRenamers(c *config.Config) *[]Renamer {
	var renamers []Renamer
	var renamer Renamer

	for _, r := range c.Renamers {
		switch r {
		case "constant":
			renamer = &constantRenamer{}
		case "static":
			renamer = &staticRenamer{}
		case "client":
			renamer = &clientRenamer{}
		case "numeric":
			renamer = &numericRenamer{}
		case "classified":
			renamer = &classifiedRenamer{}
		default:
			continue
		}

		renamer.Initialize(c)
		renamers = append(renamers, renamer)
	}

	return &renamers
}

func (r RenameHandler) ShouldHandle() bool {
	return len(r.renamers) > 0
}

func (r RenameHandler) Handle(m *monitors.Monitors) bool {
	i := -1
	for _, monitor := range *m {
		i = -1
		for _, desktop := range monitor.Desktops {
			i++
			for _, renamer := range r.renamers {
				if !renamer.CanRename(&desktop, i) {
					continue
				}

				if !renamer.Rename(&desktop, i) {
					break
				}

				return true
			}
		}
	}

	return false
}

func (c *constantRenamer) Initialize(conf *config.Config) {
	c.name = conf.Names.Constant
}

func (c constantRenamer) CanRename(desktop *monitors.Desktop, desktopIdx int) bool {
	return true
}

func (c constantRenamer) Rename(desktop *monitors.Desktop, desktopIdx int) bool {
	if desktop.Name == c.name {
		return false
	}

	err := desktop.Rename(c.name)
	if err != nil {
		log.Println("Unable to rename desktop: ", desktop.Name, err)
		return false
	}

	return true
}

func (s *staticRenamer) Initialize(conf *config.Config) {
	s.names = conf.Names.Static
}

func (s staticRenamer) CanRename(desktop *monitors.Desktop, desktopIdx int) bool {
	if desktopIdx >= len(s.names) {
		return false
	}

	return true
}

func (s staticRenamer) Rename(desktop *monitors.Desktop, desktopIdx int) bool {
	if desktop.Name == s.names[desktopIdx] {
		return false
	}

	err := desktop.Rename(s.names[desktopIdx])
	if err != nil {
		log.Println("Unable to rename desktop: ", desktop.Name, err)
		return false
	}

	return true
}

func (c *clientRenamer) Initialize(conf *config.Config) {
	c.max_length = conf.MaxLength
	c.unique_name = conf.UniqueName
}
func (c clientRenamer) CanRename(desktop *monitors.Desktop, desktopIdx int) bool {
	return len(desktop.Clients().Names(false)) > 0
}
func (c clientRenamer) Rename(desktop *monitors.Desktop, desktopIdx int) bool {
	name := strconv.Itoa(desktopIdx+1) + " "

	client_names := strings.Join(desktop.Clients().Names(c.unique_name), " ")
	if len(client_names) > c.max_length {
		runes := []rune(client_names)
		if len(runes) > c.max_length {
			client_names = fmt.Sprintf("%s..%s", string(runes[:c.max_length/2]), string(runes[len(runes)-c.max_length/2:]))
		}
	}

	name += client_names
	if desktop.Name == name {
		return false
	}

	err := desktop.Rename(name)
	if err != nil {
		log.Println("Unable to rename desktop: ", desktop.Name, err)
		return false
	}

	return true
}

func (n *numericRenamer) Initialize(conf *config.Config) {}
func (n numericRenamer) CanRename(desktop *monitors.Desktop, desktopIdx int) bool {
	return true
}
func (n numericRenamer) Rename(desktop *monitors.Desktop, desktopIdx int) bool {
	numericName := strconv.Itoa(desktopIdx + 1)

	if desktop.Name == numericName {
		return false
	}

	err := desktop.Rename(numericName)
	if err != nil {
		log.Println("Unable to rename desktop: ", desktop.Name, err)
		return false
	}

	return true
}

func (c *classifiedRenamer) Initialize(conf *config.Config) {
	c.priorityMap = make(map[string]classification)
	c.unique_name = conf.UniqueName
	priority := 0

	for _, classMap := range conf.Names.Classified {
		for name, clients := range classMap {
			for _, client := range clients {
				classification := classification{name: name, priority: priority}
				c.priorityMap[client] = classification
			}

			priority++
		}
	}
}

func (c classifiedRenamer) CanRename(desktop *monitors.Desktop, desktopIdx int) bool {
	for _, name := range desktop.Clients().Names(true) {
		if _, ok := c.priorityMap[name]; ok {
			return true
		}
	}

	return false
}

func (c classifiedRenamer) Rename(desktop *monitors.Desktop, desktopIdx int) bool {
	classifiedName := strconv.Itoa(desktopIdx + 1)

	for _, name := range desktop.Clients().Names(c.unique_name) {
		class, ok := c.priorityMap[name]
		if !ok {
			classifiedName += " " + name
		} else {
			classifiedName += " " + class.name
		}
	}

	if desktop.Name == classifiedName {
		return false
	}

	err := desktop.Rename(classifiedName)
	if err != nil {
		log.Println("Unable to rename desktop: ", desktop.Name, err)
		return false
	}

	return true
}
