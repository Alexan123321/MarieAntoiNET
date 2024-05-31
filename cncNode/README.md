
# cncNode

Similar to the botNode, the cncNode operates as a Waku light node and relies on a bootstrapNode for network access. Directly controlled by the botmaster, it serves to send command and control messages to the botNodes across the network.

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
   cncNode -botAddress <host:port> -bootstrapAddress <bootstrap_node_address>

   ```
