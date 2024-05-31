
# botNode

The botNode operates as a Waku light node, and it requires connection to the network via a bootstrapNode. It subscribes to specific topics, using the 12/WAKU2-FILTER protocol to receive commands or control messages and acts accordingly. This node can be any compromised host, (e.g., a mobile phone, a PLC, a web server, etc.). As such, a bootstrapNode can be considered an "upgraded" botNode

## Installation

To install cncNode, you need to have Go installed on your machine. You can download and install Go from the [official website](https://golang.org/dl/).

1. Clone the repository:
   ```sh
   git clone https://github.com/alexan123321/MarieAntoiNET
   ```

2. Change the project directory:
   ```sh
   cd MarieAntoiNET
   ```

3. Change the project directory:
   ```sh
   cd cncNode
   ```

4. Install the required dependencies:
   ```sh
   go mod tidy
   ```

5. Compile the code:
   ```sh
   go build .
   ```

## Usage

To run the cncNode, please use the following command:
   ```sh
   ./botNode -botAddress <host:port> -bootstrapAddress <bootstrap_node_address>

   ```

## Configuration

1. Remember to update the variable contractAddress, as this is given, once you deploy the BootstrapAddressManager.
2. Remember to update the Ethereum Sepolia client address, as the one in the code is a static one used for development purposes, and may be shut down.
