package monitors

import (
	"btops/ipc"
	"encoding/json"
	"log"
	"sort"
	"strconv"

	"github.com/mpvl/unique"
)

type bspwmState struct {
	Monitors Monitors
}

type Monitor struct {
	Name     string
	Id       int
	Desktops []Desktop
}

type Monitors []Monitor

type Desktop struct {
	Name string
	Id   int
	Root *Node
}

type Node struct {
	Id          int
	Client      *Client
	FirstChild  *Node
	SecondChild *Node
}

type Client struct {
	ClassName string
}

type Clients struct {
	clients map[string]int
}

func newClients(nodes []*Node) (clients Clients) {
	clients.clients = make(map[string]int, len(nodes))

	for _, node := range nodes {
		if node.Client == nil {
			continue
		}

		clients.clients[node.Client.ClassName]++
	}

	return clients
}

func (c Clients) Names(unique_names bool) (names []string) {
	names = make([]string, 0, len(c.clients))

	for key, count := range c.clients {
		for count > 0 {
			names = append(names, key)
			count--
		}
	}

	sort.Strings(names)

	if unique_names {
		unique.Strings(&names)
	}
	return names
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func GetMonitors(ignore []string) (*Monitors, error) {
	jsonState, err := ipc.Send("wm", "-d")
	if err != nil {
		return nil, err
	}

	var state bspwmState
	if err = json.Unmarshal(jsonState, &state); err != nil {
		return nil, err
	}

	var ms Monitors
	for _, m := range state.Monitors {
		if contains(ignore, m.Name) {
			log.Println("Ignore monitor", m.Name)
			continue
		}
		ms = append(ms, m)
	}

	state.Monitors = ms

	return &state.Monitors, nil
}

func (d Desktop) IsEmpty() bool {
	return d.Root == nil
}

func (d Desktop) Clients() (clients Clients) {
	return newClients(d.Nodes())
}

func (d Desktop) Nodes() (nodes []*Node) {
	collectNodes(d.Root, &nodes)
	return nodes
}

func collectNodes(node *Node, nodes *[]*Node) {
	if node == nil {
		return
	}

	*nodes = append(*nodes, node)
	collectNodes(node.FirstChild, nodes)
	collectNodes(node.SecondChild, nodes)
}

func (d *Desktop) Rename(name string) error {
	if _, err := ipc.Send("desktop", strconv.Itoa(d.Id), "-n", name); err != nil {
		return err
	}

	d.Name = name
	return nil
}

func (m *Monitor) AppendDesktop(name string) error {
	if _, err := ipc.Send("monitor", strconv.Itoa(m.Id), "-a", name); err != nil {
		return err
	}

	m.Desktops = append(m.Desktops, Desktop{Name: name})
	return nil
}

func (m *Monitor) RemoveDesktop(id int) error {
	if _, err := ipc.Send("desktop", strconv.Itoa(id), "-r"); err != nil {
		return err
	}

	for i := range m.Desktops {
		if m.Desktops[i].Id != id {
			continue
		}

		m.Desktops = append(m.Desktops[:i], m.Desktops[i+1:]...)
		break
	}

	return nil
}

func (m *Monitor) EmptyDesktops() (desktops []*Desktop) {
	for i := range m.Desktops {
		if !m.Desktops[i].IsEmpty() {
			continue
		}

		desktops = append(desktops, &m.Desktops[i])
	}

	return desktops
}
