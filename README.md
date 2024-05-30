## Terminal Chat Application using Golang and gRPC

This is a simple chat application which can be run in the terminal.

- Supports creating users and chatrooms.
- One user can only be registered to one chat room for now.
- Users in the same chat room can exchange messages.
- Chat history is not available, i.e., users can only view the messages which arrived after they have joined the chat room.

### To run the application

1. Clone the repository.
2. First get the server up by running (windows):

```
cd server
go build server.go
server.exe
```

3. Get a couple of clients up by running these commands in separate terminals (windows):

```
cd client
go build client.go
client.exe -user=<User Name> -chatroom=<Chat Room Name>
```

where you can enter any combination of User Name and Chat Room Name.

Now you can have a chat between the terminals within the same chat room!!!

---

To close the application, you can press Ctrl + C in the clients first, and then in the server.

---

**Note - Only users in the same named chat room can chat between themselves.**\
**Note 2 - Multiple users can have same name. But chat room names need to be unique.**


### Hope you enjoy this project :) Let me know of you learned something from this project.