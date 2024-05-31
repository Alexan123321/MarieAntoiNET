package main

import (
	"context"      // Provides functionalities for managing deadlines, cancellation signals, and other request-scoped values across API boundaries.
	"crypto/ecdsa" // Provides cryptographic operations using elliptic curves.
	"crypto/rand"  // Provides functions for generating cryptographically secure random numbers.
	"encoding/hex" // Provides functions for encoding and decoding data in hexadecimal format.
	"flag"
	"fmt"       // Provides I/O formatting functions.
	"net"       // Provides interfaces for network I/O, including TCP/IP, UDP, etc.
	"os"        // Provides platform-independent interface to operating system functionality.
	"os/signal" // Provides functionality to handle incoming signals from the operating system.
	"syscall"   // Provides an interface to the low-level operating system primitives.

	"github.com/ethereum/go-ethereum/crypto"   // Ethereum-specific cryptographic libraries for ECDSA.
	logging "github.com/ipfs/go-log/v2"        // Advanced logging library used by IPFS, supports different logging levels.
	"github.com/waku-org/go-waku/waku/v2/node" // Waku protocol version 2 node functionalities.
	"github.com/waku-org/go-waku/waku/v2/payload"
	"github.com/waku-org/go-waku/waku/v2/protocol"
)

// LogLevel for categorizing the severity of the logs.
type LogLevel int

const (
	LogInfo LogLevel = iota
	LogError
	LogFatal
)

var (
	log         = logging.Logger("fullnode")
	pubSubTopic = protocol.DefaultPubsubTopic{}
	addressFlag = flag.String("address", "0.0.0.0:60000", "Address to bind the Waku node")
)

// main is the entry point for the program.
func main() {
	// Parse command-line flags
	address := parseCommandLineArguments()
	// Resolve the network address to a TCP address structure.
	hostAddr := resolveTCPAddress(address)
	// Generate an ECDSA private key for cryptographic operations.
	prvKey := generatePrivateKey()

	// Create a context with cancellation capabilities.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure that the cancel function is called to clean up resources.

	// Initialize and configure a new Waku node.
	fullNode := initializeNode(hostAddr, prvKey)
	// Start the Waku node.
	startNode(ctx, fullNode)
	// Print the listening addresses of the node.
	printNodeAddresses(fullNode)

	// Start a goroutine that continuously reads messages from the network.
	go readLoop(ctx, fullNode)

	// Wait for a shutdown signal to cleanly terminate the program.
	waitForShutdownSignal()
	// Stop the node and clean up resources.
	shutdownNode(fullNode)
}

// parseCommandLineArguments retrieves the network address from command line arguments using flag package.
func parseCommandLineArguments() string {
	flag.Parse()        // Parses the command-line flags
	return *addressFlag // Returns the parsed address
}

// resolveTCPAddress converts a string network address into a *net.TCPAddr.
func resolveTCPAddress(address string) *net.TCPAddr {
	// Attempt to resolve the TCP address.
	hostAddr, err := net.ResolveTCPAddr("tcp", address)
	// Check for errors and terminate if the address cannot be resolved.
	if err != nil {
		log.Fatal("Could not resolve TCP address: ", err)
	}
	return hostAddr
}

// generatePrivateKey creates a new ECDSA private key for use in cryptographic operations.
func generatePrivateKey() *ecdsa.PrivateKey {
	// Generate a random 32-byte hexadecimal string.
	key, err := randomHex(32)
	checkError(err, "Could not generate random key", LogFatal)
	// Convert the hexadecimal string into an ECDSA private key.
	prvKey, err := crypto.HexToECDSA(key)
	checkError(err, "Invalid key", LogFatal)
	return prvKey
}

// initializeNode configures and initializes a new Waku node.
func initializeNode(hostAddr *net.TCPAddr, prvKey *ecdsa.PrivateKey) *node.WakuNode {
	// Create a new Waku node with the specified private key and host address.
	fullNode, err := node.New(
		node.WithPrivateKey(prvKey),
		node.WithHostAddress(hostAddr),
		node.WithWakuRelay(),
		node.WithWakuFilterFullNode(),
	)
	// Check for initialization errors and terminate if necessary.
	checkError(err, "Error initializing the node", LogFatal)
	return fullNode
}

// startNode starts the Waku node and begins its network operations.
func startNode(ctx context.Context, fullNode *node.WakuNode) {
	// Start the node and check for errors.
	err := fullNode.Start(ctx)
	checkError(err, "Error starting the node", LogFatal)
}

// printNodeAddresses logs the listening addresses of the Waku node.
func printNodeAddresses(fullNode *node.WakuNode) {
	// Iterate through all listening addresses and print each one.
	for _, addr := range fullNode.ListenAddresses() {
		fmt.Println("Full node listen address:", addr)
	}
}

// waitForShutdownSignal waits for a system signal before initiating shutdown.
func waitForShutdownSignal() {
	// Set up a channel to receive system signals.
	ch := make(chan os.Signal, 1)
	// Listen for interrupt or terminate signals.
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	// Block until a signal is received.
	<-ch
	fmt.Println("Received signal, shutting down...")
}

// shutdownNode stops the Waku node and performs any necessary cleanup.
func shutdownNode(fullNode *node.WakuNode) {
	// Stop the node's operation.
	fullNode.Stop()
	fmt.Println("Node has been stopped.")
}

// randomHex generates a random hexadecimal string of a specified length.
func randomHex(n int) (string, error) {
	// Allocate a byte slice.
	bytes := make([]byte, n)
	// Read cryptographically secure random bytes.
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("could not read random bytes: %w", err)
	}
	// Return the byte slice encoded as a hexadecimal string.
	return hex.EncodeToString(bytes), nil
}

// readLoop continuously reads messages from the Waku network.
func readLoop(ctx context.Context, wakuNode *node.WakuNode) {
	// Retrieve the string representation of the pubSubTopic.
	pubsubTopic := pubSubTopic.String()
	// Subscribe to the relay service with a content filter.
	sub, err := wakuNode.Relay().Subscribe(ctx, protocol.NewContentFilter(pubsubTopic))
	checkError(err, "Could not subscribe", LogError)
	// Check for subscription errors and log if an issue occurs.
	if err != nil {
		checkError(err, "Failed to decode the message", LogError)
		return
	}

	// Process incoming messages in a loop.
	for value := range sub[0].Ch {
		// Decode the message payload.
		_, err := payload.DecodePayload(value.Message(), &payload.KeyInfo{Kind: payload.None})
		// If an error occurs in decoding, log the error and return.
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}

// checkError evaluates an error and handles it according to a specified LogLevel.
func checkError(err error, message string, level LogLevel) {
	if err != nil {
		switch level {
		case LogInfo:
			log.Info(message, err)
		case LogError:
			log.Error(message, err)
		case LogFatal:
			log.Fatalf("%s: %v", message, err)
		default:
			log.Error("Unhandled log level", "provided level", level)
		}
	}
}
