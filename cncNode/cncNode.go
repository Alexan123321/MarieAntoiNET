package main

import (
	"bufio"        // Provides buffered I/O utilities, useful for reading/writing data in buffered chunks.
	"context"      // Provides context management functionalities, useful for deadline/cancellation propagation.
	"crypto/ecdsa" // Provides functionalities for working with ECDSA cryptographic keys.
	"crypto/rand"  // Provides functionalities to generate cryptographically secure random numbers.
	"encoding/hex" // Provides functions for encoding and decoding hexadecimal data.
	"flag"
	"fmt"       // Provides I/O formatting functions for formatted I/O operations.
	"net"       // Provides network I/O functionalities, for working with TCP/IP and other protocols.
	"os"        // Provides platform-independent interface to operating system functionalities.
	"os/signal" // Provides utilities to handle incoming operating system signals.
	"strings"   // Provides utilities for string manipulation.
	"syscall"   // Provides an interface for low-level operating system calls.

	"github.com/ethereum/go-ethereum/crypto"                    // Ethereum's cryptographic library for operations like ECDSA.
	logging "github.com/ipfs/go-log/v2"                         // Advanced logging library used by IPFS, supports different logging levels.
	"github.com/multiformats/go-multiaddr"                      // Library to handle multi-format addresses, useful in peer-to-peer networks.
	"github.com/waku-org/go-waku/waku/v2/node"                  // Waku protocol version 2 node functionalities.
	"github.com/waku-org/go-waku/waku/v2/payload"               // Manages payloads within the Waku protocol.
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"         // Manages peer storage, used in maintaining peer information.
	"github.com/waku-org/go-waku/waku/v2/protocol"              // Contains definitions and utilities related to the Waku protocol.
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"       // Manages message filtering in Waku protocol.
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"           // Protocol buffers for Waku, used for structured data serialization.
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"        // Manages message relaying in the Waku network.
	"github.com/waku-org/go-waku/waku/v2/protocol/subscription" // Manages subscriptions to topics in the Waku network.
	"github.com/waku-org/go-waku/waku/v2/utils"                 // Provides utility functions used across Waku protocol implementations.
	"google.golang.org/protobuf/proto"                          // Google's protocol buffers package, used for serializing structured data.
)

// Declaring global variables to hold application-wide configurations and state.
var (
	log              = logging.Logger("lightnode")                                                                                                   // Logger instance for logging across the application.
	pubSubTopic      = protocol.DefaultPubsubTopic{}                                                                                                 // Default pub/sub topic for Waku messages.
	symKey           = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32} // Symmetric key for cryptographic operations.
	privateKey       = "1fc8abf855c6d5e97bdb8d7df8670c7f3f71521dd3754be4050b8b5bde0fc35d"                                                            // Hardcoded private key for the node (should be secured).
	botAddress       = flag.String("botAddress", "0.0.0.0:60002", "Address for the bot node")                                                        // Flag for specifying the bot's network address.
	bootstrapAddress = flag.String("bootstrapAddress", "", "Address for the bootstrap node")                                                         // Flag for specifying the bootstrap node's address.
)

// LogLevel defines levels of logging based on severity and action required.
type LogLevel int

const (
	LogInfo      LogLevel                                               = iota // LogInfo is used for non-critical informational messages.
	LogError                                                                   // LogError is used for standard error logging.
	LogFatal                                                                   // LogFatal is used for errors that require the application to terminate.
	contentTopic = "/masterThesis/1/command-control-registration/proto"        // Constant for defining a specific content topic for messages.
)

func main() {
	// Initialize command-line flags.
	flag.Parse()

	// Use parseCommandLineArgs to get bot and bootstrap addresses.
	botAddress, bootstrapAddress := parseCommandLineArgs()

	// Ensure the bootstrap address is provided, otherwise log a fatal error.
	if bootstrapAddress == "" {
		log.Fatal("Bootstrap address must be provided. Usage: ./botNode -botAddress=[optional] -bootstrapAddress=[required]")
	}

	// Resolve the TCP address from the bot address.
	hostAddr := resolveTCPAddr(botAddress)
	// Generate an ECDSA private key.
	prvKey := generatePrivateKey()

	// Create a context with cancellation capabilities to manage the lifecycle of the application.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup and initialize the Waku node.
	lightNode := setupNode(ctx, hostAddr, prvKey)
	defer lightNode.Stop()

	// Subscribe to other nodes using the bootstrap address.
	filter := subscribeToNode(ctx, lightNode, bootstrapAddress)
	// Listen for messages based on the established filters.
	listenForMessages(ctx, filter)
	// Run a loop to handle writing and sending messages.
	writeLoop(ctx, lightNode)
	// Wait for a signal to shutdown.
	waitForSignalAndShutdown()
	// Clean up and shut down the node.
	shutdownNode(lightNode)
}

func parseCommandLineArgs() (string, string) {
	// Returns the parsed command line arguments for bot and bootstrap addresses.
	return *botAddress, *bootstrapAddress
}

func resolveTCPAddr(address string) *net.TCPAddr {
	// Resolve a string address to a TCPAddr. Fatal error on failure.
	hostAddr, err := net.ResolveTCPAddr("tcp", address)
	checkError(err, "Could not resolve TCP address", LogFatal)
	return hostAddr
}

func generatePrivateKey() *ecdsa.PrivateKey {
	// Generate a random private key for ECDSA. Fatal error on failure.
	key, err := randomHex(32)
	checkError(err, "Could not generate random key", LogFatal)
	prvKey, err := crypto.HexToECDSA(key)
	checkError(err, "Invalid key", LogFatal)
	return prvKey
}

// setupNode initializes and starts a Waku node with the given configurations.
func setupNode(ctx context.Context, hostAddr *net.TCPAddr, prvKey *ecdsa.PrivateKey) *node.WakuNode {
	// Create a new Waku node with specified configurations:
	// Private key, host address, relay and filter functionalities enabled.
	lightNode, err := node.New(
		node.WithPrivateKey(prvKey),    // Sets the private key for node identity.
		node.WithHostAddress(hostAddr), // Sets the TCP/IP address for the node.
		node.WithWakuRelay(),           // Enables Waku Relay protocol for message passing.
		node.WithWakuFilterLightNode(), // Enables filtering capabilities for the node.
	)
	// Use checkError to handle initialization errors, with LogFatal to terminate if failure occurs.
	checkError(err, "Error initializing node", LogFatal)

	// Start the node using the provided context to manage its lifecycle.
	err = lightNode.Start(ctx)
	// Handle errors that occur during node start, also with LogFatal.
	checkError(err, "Error starting node", LogFatal)

	// Return the fully initialized and running Waku node.
	return lightNode
}

// subscribeToNode connects the node to a bootstrap peer and subscribes to a content filter.
func subscribeToNode(ctx context.Context, lightNode *node.WakuNode, bootstrapAddress string) []*subscription.SubscriptionDetails {
	// Parse the bootstrap address into a multiaddr format.
	bootstrapMultiAddress, err := multiaddr.NewMultiaddr(bootstrapAddress)
	// Use checkError to handle parsing errors, logging fatally if the address is incorrect.
	checkError(err, "Error parsing bootstrap node multiaddr", LogFatal)

	// Add the bootstrap node as a peer with a static subscription.
	_, err = lightNode.AddPeer(bootstrapMultiAddress, wps.Static, []string{pubSubTopic.String()}, filter.FilterSubscribeID_v20beta1)
	// Handle errors from adding a peer, which is critical for node connectivity.
	checkError(err, "Error adding filter peer on light node", LogFatal)

	// Define a content filter for the subscription.
	cf := protocol.ContentFilter{
		PubsubTopic:   relay.DefaultWakuTopic,                    // Default topic for Waku network.
		ContentTopics: protocol.NewContentTopicSet(contentTopic), // Custom content topics for filtering.
	}

	// Subscribe the node to the network with the defined content filter.
	filter, err := lightNode.FilterLightnode().Subscribe(ctx, cf)
	// Handle subscription errors, logging fatally as this step is crucial for receiving messages.
	checkError(err, "Error subscribing", LogFatal)

	// Return the subscription details, which contain channels to receive filtered messages.
	return filter
}

// listenForMessages listens asynchronously for messages from a given subscription and handles them.
func listenForMessages(ctx context.Context, filter []*subscription.SubscriptionDetails) {
	go func() { // Start a new goroutine to handle messages independently of the main execution.
		fmt.Println("Listening for messages via filter...")

		for { // Infinite loop to continuously listen for messages.
			select {
			case <-ctx.Done(): // Check if the context has been cancelled or expired.
				return // Exit the goroutine if the context is done.
			case env := <-filter[0].C: // Receive a message from the subscription channel.
				if env.Message() == nil {
					continue // If no message is present, skip to the next iteration.
				}

				// Attempt to decode the message using symmetric key information.
				messagePayload, err := payload.DecodePayload(env.Message(), &payload.KeyInfo{
					Kind:   payload.Symmetric,
					SymKey: symKey,
				})

				// If an error occurs during decoding with symmetric key.
				if err != nil {
					log.Error("Failed to decode message with symmetric key: ", err)
					// Attempt decoding with an asymmetric key as a fallback.
					privateKey, err := crypto.HexToECDSA(privateKey)
					checkError(err, "Failed to convert hex to ECDSA", LogError) // Handle key conversion error.

					// Try decoding with the asymmetric key.
					messagePayload, err = payload.DecodePayload(env.Message(), &payload.KeyInfo{
						Kind:    payload.Asymmetric,
						PrivKey: privateKey,
					})
					checkError(err, "Failed to decode message with asymmetric key", LogError) // Handle decoding error.
				}

				// Process the successfully decoded message.
				if messagePayload != nil {
					fmt.Printf("Filtered message received: %s\n", string(messagePayload.Data))
				}
			}
		}
	}()
}

// waitForSignalAndShutdown blocks execution until an OS interrupt or termination signal is received.
func waitForSignalAndShutdown() {
	ch := make(chan os.Signal, 1) // Create a channel to receive OS signals.

	// Notify the channel of interrupt and terminate signals.
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received.
	<-ch

	// Log receipt of the signal.
	fmt.Println("Received signal, shutting down...")
}

// shutdownNode stops the Waku node and logs its cessation.
func shutdownNode(fullNode *node.WakuNode) {
	fullNode.Stop()                       // Stop the node's operations cleanly.
	fmt.Println("Node has been stopped.") // Log that the node has stopped.
}

// randomHex generates a random hexadecimal string of the specified length.
func randomHex(n int) (string, error) {
	bytes := make([]byte, n) // Allocate a byte slice of size n.

	// Attempt to fill the byte slice with cryptographically secure random bytes.
	if _, err := rand.Read(bytes); err != nil {
		// Return an empty string and the error if random byte generation fails.
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Encode the bytes to a hexadecimal string and return it.
	return hex.EncodeToString(bytes), nil
}

// write prepares and sends an encrypted message using the WakuNode's communication facilities.
func write(ctx context.Context, wakuNode *node.WakuNode, msgContent string) {
	// Define a constant message version for compatibility.
	var version uint32 = 1

	// Convert the private key from hex to ECDSA, ignoring errors (improvement needed).
	privateKey, err := crypto.HexToECDSA(privateKey)
	checkError(err, "Failed to convert hex to ECDSA private key", LogError) // Use checkError to handle conversion errors.

	// Create a new payload structure to hold the message content.
	p := new(payload.Payload)
	// Combine node ID with the message content to form the payload data.
	p.Data = []byte(wakuNode.ID() + ": " + msgContent)
	// Set both symmetric and private keys in the payload for encryption.
	p.Key = &payload.KeyInfo{
		Kind:    payload.Symmetric,
		SymKey:  symKey,
		PrivKey: privateKey,
	}

	// Encode the payload into an encrypted format.
	encryptedPayload, err := p.Encode(version)
	// Use checkError to log encryption errors and return early if an error occurs.
	checkError(err, "Error encrypting the message", LogError)

	// Create a Waku message structure to hold the encrypted payload and metadata.
	msg := &pb.WakuMessage{
		Payload:      encryptedPayload,
		Version:      proto.Uint32(version),
		ContentTopic: contentTopic,
		Timestamp:    utils.GetUnixEpoch(wakuNode.Timesource()),
	}

	// Sign the message using the private key and the defined pub/sub topic.
	relay.SignMessage(privateKey, msg, pubSubTopic.String())

	// Publish the signed message to the Waku network.
	wakuNode.Relay().Publish(ctx, msg, relay.WithPubSubTopic(pubSubTopic.String()))
}

// writeLoop continuously reads messages from standard input and sends them using the write function.
func writeLoop(ctx context.Context, wakuNode *node.WakuNode) {
	// Create a new buffered reader for standard input.
	reader := bufio.NewReader(os.Stdin)

	// Prompt the user for input.
	fmt.Println("Enter messages to send, type 'exit' to stop:")

	for {
		fmt.Print("Message: ") // Prompt for individual messages.

		// Read a line of input from the user.
		input, err := reader.ReadString('\n')
		// Handle errors from reading input, logging them and continuing to the next iteration.
		if err != nil {
			log.Error("Failed to read from stdin: ", err)
			continue
		}

		// Trim newline and whitespace characters from the input.
		input = strings.TrimSpace(input)

		// Check if the user wants to exit the loop.
		if input == "exit" {
			fmt.Println("Exiting write loop...")
			return
		}

		// Send the trimmed input as a message.
		write(ctx, wakuNode, input)
	}
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
