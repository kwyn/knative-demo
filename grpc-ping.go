// +build grpcping

package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	ping "github.com/kwyn/knative-demo/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var port = 8080

type pingServer struct {
	flavor string
}

func (p *pingServer) Ping(ctx context.Context, req *ping.Request) (*ping.Response, error) {
	return &ping.Response{Msg: fmt.Sprintf("%s - pong (flavor: %s)", req.Msg, p.flavor)}, nil
}

func (p *pingServer) PingStream(stream ping.PingService_PingStreamServer) error {
	for {
		req, err := stream.Recv()

		if err == io.EOF {
			fmt.Println("Client disconnected")
			return nil
		}

		if err != nil {
			fmt.Println("Failed to receive ping")
			return err
		}

		fmt.Printf("Replying to ping %s at %s\n", req.Msg, time.Now())

		err = stream.Send(&ping.Response{
			Msg: fmt.Sprintf("pong %s", time.Now()),
		})

		if err != nil {
			fmt.Printf("Failed to send pong %s\n", err)
			return err
		}
	}
}

func main() {
	fmt.Println("Server starting...")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	flavor := os.Getenv("FLAVOR")
	fmt.Println("flavor", flavor)
	pingServer := &pingServer{flavor}

	// The grpcServer is currently configured to serve h2c traffic by default.
	// To configure credentials or encryption, see: https://grpc.io/docs/guides/auth.html#go
	grpcServer := grpc.NewServer()
	ping.RegisterPingServiceServer(grpcServer, pingServer)
	fmt.Printf("Server gonna listen: %v", port)
	err = grpcServer.Serve(lis)
	if err != nil {
		log.Fatalf("Error occured +V%", err)
	}
}
