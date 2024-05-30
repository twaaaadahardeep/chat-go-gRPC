package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/twaaaadahardeep/chat-go-gRPC/proto"
	"google.golang.org/grpc"
)

var (
	addr = flag.Int("addr", 8081, "Address where the server will listen to. (default = 8081)")
)

type server struct {
	proto.UnimplementedChatServiceServer
	mu             sync.Mutex
	chatRooms      map[string]*proto.ChatRoom
	users          map[string]*proto.User
	streamMappings map[string]proto.ChatService_ChatServer
}

func newServer() *server {
	return &server{
		chatRooms:      make(map[string]*proto.ChatRoom),
		users:          make(map[string]*proto.User),
		streamMappings: make(map[string]proto.ChatService_ChatServer),
	}
}

func (s *server) Chat(chatStream proto.ChatService_ChatServer) error {
	for {
		in, err := chatStream.Recv()
		fmt.Printf("\nMessage received: %v\n", in)
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		userExists := false

		user := in.User
		chatRoom := in.ChatRoom

		s.mu.Lock()

		// check for the chat room
		currChatRoom, ok := s.chatRooms[chatRoom.ChannelId]
		if !ok {
			fmt.Println("couldn't find the chatroom...")
		}
		fmt.Printf("\nchatRoom in use: %v\n", currChatRoom)

		// check if user exists in the current chat room
		for _, v := range currChatRoom.Users {
			fmt.Printf("iterating chat room users: %v\n", v)
			if user.UserId == v.UserId {
				userExists = true
				break
			}
		}

		// if user doesn't exist in the chat room, then add it to the chat room
		if !userExists {
			fmt.Printf("appending user: %v to current chat room\n", user)
			currChatRoom.Users = append(currChatRoom.Users, user)
		}

		// check if the user exists in the stream mapping
		_, senderExists := s.streamMappings[user.UserId]

		// if user is not present then add the user stream to the stream mappings
		if !senderExists {
			fmt.Printf("appending user stream: %v to user %v\n", chatStream, user)
			s.streamMappings[user.UserId] = chatStream
		}

		// now the user is added to the chat room and the user stream is mapped properly
		s.mu.Unlock()

		// iterate over the chat room users
		for _, currUser := range currChatRoom.Users {

			// is the user == current user, skip sending the message
			if currUser.UserId == user.UserId {
				fmt.Printf("skipping user: %v\n", user)
				continue
			}

			// get the chat stream for the user
			recipientStream, ok := s.streamMappings[currUser.UserId]
			if !ok {
				fmt.Printf("user not found... recipientStream\n")
				continue
			}

			fmt.Printf("stream %v found for user %v\n", recipientStream, currUser)

			// send the message to the chat stream of the user
			fmt.Printf("\nchat sent to: %v\n", currUser)
			if err := recipientStream.Send(in); err != nil {
				fmt.Printf("Error sending message: %v", err)
			}

			fmt.Printf("successfully sent the message... %v", in)
		}
	}
}

func (s *server) CreateChatRoom(ctx context.Context, chatRoom *proto.ChatRoom) (*proto.ChatRoom, error) {
	for _, v := range s.chatRooms {
		if v.GetChannelName() == chatRoom.GetChannelName() {
			return nil, errors.New("chat room with the same Channel Name exists")
		}
	}

	s.mu.Lock()
	s.chatRooms[chatRoom.ChannelId] = chatRoom
	s.mu.Unlock()

	log.Printf("chatRooms: %v", s.chatRooms)

	return chatRoom, nil
}

func (s *server) CreateUser(ctx context.Context, user *proto.User) (*proto.User, error) {
	for _, u := range s.users {
		if u.GetUserId() == user.GetUserId() {
			return nil, errors.New("user with the same userId exists")
		}
	}

	s.mu.Lock()
	s.users[user.UserId] = user
	s.mu.Unlock()

	log.Printf("users: %v", s.users)

	return user, nil
}

func (s *server) GetChatRooms(e *proto.Empty, stream proto.ChatService_GetChatRoomsServer) error {
	for _, v := range s.chatRooms {
		if err := stream.Send(v); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *addr))
	if err != nil {
		log.Fatalf("Couldn't start server: %v", err)
	}
	defer lis.Close()

	s := grpc.NewServer()

	server := *newServer()
	proto.RegisterChatServiceServer(s, &server)

	log.Printf("Listening on port: %v", lis.Addr())

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
