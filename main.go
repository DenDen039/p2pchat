package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func handleMessage(conn net.Conn, name string) {
	reader := bufio.NewReader(conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Connection with %s lost.\n", conn.RemoteAddr().String())
			conn.Close()
			break
		}
		params := strings.Split(name, ":")

		fmt.Printf("\n%s(%s): %s", params[0], params[1], message)
		fmt.Print("Enter command or message: ")
	}
}

func SendNameToPeer(name string, conn net.Conn) error {
	writer := bufio.NewWriter(conn)
	_, err := writer.WriteString("/name " + name + "\n")
	writer.Flush()
	return err
}

func RecieveNameFromPeer(conn net.Conn) (string, error) {
	reader := bufio.NewReader(conn)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	command := strings.TrimSpace(input)

	if strings.HasPrefix(command, "/name ") {
		params := strings.Split(command, " ")
		if len(params) != 2 {
			return "", fmt.Errorf("invalid name command")
		}
		return params[1] + ":" + strings.Split(conn.RemoteAddr().String(), ":")[1], nil
	}

	return "", fmt.Errorf("invalid name command")
}

func AddPeer(conn net.Conn, peers map[string]net.Conn, hostName string) (string, error) {
	err := SendNameToPeer(hostName, conn)
	if err != nil {
		conn.Close()
		return "", fmt.Errorf("error cannot send name to peer %s err: %s", conn.RemoteAddr().String(), err)
	}

	name, err := RecieveNameFromPeer(conn)
	if err != nil {
		conn.Close()
		return "", fmt.Errorf("error recieving name from peer %s err: %s", conn.RemoteAddr().String(), err)
	}

	fmt.Printf("\nNew connection from %s. Name: %s\n", conn.RemoteAddr().String(), name)
	peers[name] = conn

	return name, nil
}

func RemovePeer(name string, peers map[string]net.Conn) error {
	conn, ok := peers[name]
	if !ok {
		return fmt.Errorf("no peer found with name %s", name)
	}
	delete(peers, name)
	conn.Close()
	return nil
}

func SendMessageToRecievers(message string, recievers []string, peers map[string]net.Conn) error {
	for _, name := range recievers {
		_, ok := peers[name]
		if !ok {
			fmt.Printf("No peer found with name %s\n", name)
			continue
		}
		_, err := peers[name].Write([]byte(message + "\n"))
		if err != nil {
			return fmt.Errorf("error sending message to peer: %s", err)
		}
	}
	return nil
}

func reciever(peers map[string]net.Conn, listener *net.Listener, hostName string) {
	for {
		conn, err := (*listener).Accept()
		if err != nil {
			fmt.Print("Error accepting connection:", err)
			fmt.Print("\nEnter command or message: ")
			continue
		}

		name, err := AddPeer(conn, peers, hostName)
		if err != nil {
			fmt.Println("Error: ", err)
			fmt.Print("\nEnter command or message: ")
			continue
		}
		fmt.Print("\nEnter command or message: ")
		go handleMessage(conn, name)
	}
}

func sender(peers map[string]net.Conn, listener *net.Listener, hostAddress, hostName string) {
	var recievers []string
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter command or message: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if strings.HasPrefix(input, "/connect ") {
			address := strings.TrimSpace(strings.TrimPrefix(input, "/connect "))
			if ":"+hostAddress == address {
				fmt.Println("Error cannot connect to self")
				continue
			}

			conn, err := net.Dial("tcp", address)
			if err != nil {
				fmt.Println("Error connecting to address:", err)
				continue
			}

			name, err := AddPeer(conn, peers, hostName)
			if err != nil {
				fmt.Println(err)
			}

			go handleMessage(conn, name)

		} else if strings.HasPrefix(input, "/disconnect ") {
			name := strings.TrimPrefix(input, "/disconnect ")
			err := RemovePeer(name, peers)
			if err != nil {
				fmt.Println("Error ", err)
			}
		} else if strings.HasPrefix(input, "/recievers ") {
			namesString := strings.TrimPrefix(input, "/recievers ")
			if namesString == "" {
				recievers = []string{}
				continue
			}
			names := strings.Split(namesString, " ")

			new_recievers := []string{}
			ok := true

			for _, name := range names {
				_, ok := peers[name]
				if !ok {
					fmt.Printf("No peer found with name %s\n", name)
					break
				}
				new_recievers = append(new_recievers, name)
			}
			if ok {
				recievers = new_recievers
				fmt.Println("Recievers updated")
			}
		} else if input == "/exit" {
			break
		} else {
			err := SendMessageToRecievers(input, recievers, peers)
			if err != nil {
				fmt.Println("Error ", err)
			}
		}
	}
}

func main() {

	peers := make(map[string]net.Conn)
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter your name: ")
	input, _ := reader.ReadString('\n')
	hostName := strings.TrimSpace(input)
	if hostName == "" {
		fmt.Println("Error name cannot be empty")
		return
	}

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		fmt.Println("Error setting up listener:", err)
		return
	}
	defer listener.Close()
	fmt.Printf("Your chat address is: %s\n", listener.Addr().String())

	// Spin up reciever
	go reciever(peers, &listener, hostName)

	// Spin up sender
	sender(peers, &listener, strings.Split(listener.Addr().String(), ":")[3], hostName)
	for _, peer := range peers {
		peer.Close()
	}
}
