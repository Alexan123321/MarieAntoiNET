
# bootstrapNode

The bootstrapNode operates as a Waku full node on a compromised server with a public IP (e.g. a web server). The purpose of the bootstrapNode is to
provide light nodes with a point of entry into the network.

## Installation

To install bootstrapNode, you need to have Go installed on your machine. You can download and install Go from the [official website](https://golang.org/dl/).

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
   cd bootstrapNode
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

To run the bootstrapNode, please use the following command:
   ```sh
   ./boostrapNode -address <host:port>
   ```
