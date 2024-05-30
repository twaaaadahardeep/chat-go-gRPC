package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/twaaaadahardeep/chat-go-gRPC/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	addr     = flag.String("addr", "localhost:8081", "the address where the client will send their requests to")
	user     = flag.String("user", "Default", "user name")
	chatRoom = flag.String("chatroom", "Default", "the chat room where the user wants to send message to. Creates a new channel if it doesn't exist.")
	userUuid = uuid.New().String()

	createdUser     *proto.User
	createdChatRoom *proto.ChatRoom
	chRooms         = make([]*proto.ChatRoom, 0, 10)
)

func init() {

}

func runChat(c proto.ChatServiceClient) {
	scanner := bufio.NewScanner(os.Stdin)

	client, err := c.Chat(context.Background())
	if err != nil {
		fmt.Printf("error fetching chat client... %v\n", err)
		return
	}

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()

		for {
			fmt.Print("Message: \n")
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					fmt.Printf("error reading input... %v", err)
				}
				return
			}

			text := scanner.Text()

			if err := client.Send(&proto.Message{
				User:     createdUser,
				ChatRoom: createdChatRoom,
				Content: &proto.Content{
					Msg: text,
				},
			}); err != nil {
				fmt.Printf("error sending the message... %v\n", err)
			}
		}
	}()

	go func() {
		defer wg.Done()

		for {
			msg, err := client.Recv()

			if err == io.EOF {
				fmt.Println("server closed the connection...")
				return
			}
			if err != nil {
				fmt.Printf("error receiving message... %v\n", err)
				return
			}

			fmt.Printf("%v: %v\n", msg.User.UserName, msg.Content.Msg)
			fmt.Print("Message: \n")
		}
	}()

	wg.Wait()
}

func createChatRoom(c proto.ChatServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour*1)
	defer cancel()

	getChatRooms(c)

	for _, v := range chRooms {
		if v.GetChannelName() == *chatRoom {
			createdChatRoom = v
			fmt.Printf("chatroom created: %v", createdChatRoom)
			return
		}
	}

	chRoom, err := c.CreateChatRoom(ctx, &proto.ChatRoom{
		ChannelId:   uuid.New().String(),
		ChannelName: *chatRoom,
	})
	if err != nil {
		fmt.Printf("error encountered: %v", err)
		return
	}

	createdChatRoom = chRoom
	log.Printf("chatroom created: %v", createdChatRoom)
}

func getChatRooms(c proto.ChatServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour*1)
	defer cancel()

	stream, err := c.GetChatRooms(ctx, &proto.Empty{})
	if err != nil {
		fmt.Printf("error getting the chat rooms: %v", err)
	}

	for {
		chRoom, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			fmt.Printf("error while fetching chat rooms: %v", err)
		}

		chRooms = append(chRooms, chRoom)
	}
}

func createUser(c proto.ChatServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour*1)
	defer cancel()

	user, err := c.CreateUser(ctx, &proto.User{
		UserId:   userUuid,
		UserName: *user,
	})
	if err != nil {
		fmt.Printf("error encountered: %v", err)
		return
	}

	createdUser = user

	log.Printf("user created: %v", createdUser)
}

func main() {
	flag.Parse()

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("could not connect: %v", err)
	}
	defer conn.Close()

	c := proto.NewChatServiceClient(conn)

	var wg sync.WaitGroup

	wg.Add(1)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		log.Printf("Interrupt signal received, shutting down...")
		conn.Close()
		wg.Done()
		os.Exit(0)
	}()

	createChatRoom(c)
	createUser(c)
	runChat(c)

	wg.Wait()
}
