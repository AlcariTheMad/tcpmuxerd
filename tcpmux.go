package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"strconv"
)

type Service struct {
	Type	bool
	Host	string
	Port	int
	Path	string
	Args	[]string
}

func srv_not_available(s *net.TCPConn) {
	s.Write([]byte("- service not available at this time\n"))
	s.Close()
}

func (srv *Service) Run(s *net.TCPConn) {
	if (srv.Type == true) {
		cmd := exec.Command(srv.Path)
		stdin, err := cmd.StdinPipe()
		if (err != nil) {
			srv_not_available(s)
                        return
                }
		stdout, err := cmd.StdoutPipe()
		if (err != nil) {
			srv_not_available(s)
                        return
                }
		err = cmd.Start()
		if (err != nil) {
			srv_not_available(s)
                        return
                }
		s.Write([]byte("+ connected\n"))
		go io.Copy(s, stdout)
		go io.Copy(stdin, s)
	} else {
		d, err := net.Dial("tcp", fmt.Sprintf("%s:%d",srv.Host, srv.Port))
		if (err != nil) {
			fmt.Println(err)
			srv_not_available(s)
                        return
                }
		s.Write([]byte("+ connected\n"))
		go io.Copy(s, d)
		go io.Copy(d, s)
	}
}

func readconf() *map[string]Service {
	cf_file, err := os.Open("tcpmux.conf")
	if (err != nil) { panic(err) }
	stat, err := cf_file.Stat()
	if (err != nil) { panic(err) }
	size := stat.Size()
	rawconf := make([]byte, size)
	cf_file.Read(rawconf)
	conf := make(map[string]Service, 1)
	lines := strings.Split(string(rawconf), "\n")
	for _, line := range lines {
		if (len(line) < 5) {continue}
		l := strings.Split(line, "\t")
		srv_entry := Service{}
		if (l[1] == "!") {
			srv_entry.Type = true
			srv_entry.Path = l[2]
			if (len(l) == 4) {
				srv_entry.Args = strings.Split(l[3], " ")
			} else {
				srv_entry.Args = make([]string, 0)
			}
		} else if (l[1] == "#") {
			var x int
			if (len(l) == 3) {
				srv_entry.Host = "localhost"
				x = 2
			} else if (len(l) == 4) {
				srv_entry.Host = l[2]
				x = 3
			}
			port, err := strconv.ParseInt(l[x], 10, 16)
			if (err != nil) { panic(err) } // TODO: print a better error
			srv_entry.Port = int(port)
		} else {
			fmt.Println("Bad tcpmux.conf, exiting...")
		}
		conf[strings.ToUpper(l[0])] = srv_entry
	}
	return &conf
}

func process(s *net.TCPConn, conf *map[string]Service) {
	srv := make([]byte, 100)
	_, err := s.Read(srv)
	if (err != nil) {
			panic(err)
	}
	service := strings.Split(strings.ToUpper(string(srv)), "\n")[0]
	//fmt.Printf("'%s'", service)
	if (service == "HELP") {
		for name, _ := range *conf {
			s.Write([]byte(fmt.Sprintf("%s\n", name)))
		}
		s.Close()
		return
	}
	srv_entry, exists := (*conf)[string(service)]
	if (exists) {
		srv_entry.Run(s)
	} else { 
		s.Write([]byte("- service not found\n"))
		s.Close()
	}
}

func main() {
	fmt.Println("TCPMUX starting up")
	conf := readconf()
	sock, err := net.ListenTCP("tcp", &net.TCPAddr{net.IPv4(0,0,0,0), 1})
	if (err != nil) { panic(err) }
	defer func(){ sock.Close() }()
	for {
		conn, err := sock.AcceptTCP()
		if (err != nil) { continue }
		go process(conn, conf)
	}
}
