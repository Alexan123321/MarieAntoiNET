# BootstrapAddressManager Deployment Guide

This guide explains how to deploy the `BootstrapAddressManager.sol` smart contract on the Ethereum Sepolia testnet using the REMIX IDE and how to use the contract address in a different node (botNode).

## Prerequisites

- MetaMask Wallet
- Sepolia Testnet ETH (you can get some from a faucet)
- REMIX IDE (https://remix.ethereum.org)

## Steps to Deploy the Smart Contract

1. **Open REMIX IDE:**
   - Go to [REMIX IDE](https://remix.ethereum.org) in your web browser.

2. **Create a New File:**
   - In the file explorer, click on the "contracts" folder.
   - Click the "+" icon to create a new file.
   - Name the file `BootstrapAddressManager.sol`.

3. **Copy and Paste the Smart Contract Code:**
   - Copy the code of `BootstrapAddressManager.sol` and paste it into the newly created file in REMIX.

4. **Compile the Smart Contract:**
   - In the left sidebar, click on the "Solidity Compiler" tab (second icon from the top).
   - Ensure the compiler version matches the pragma version in your contract.
   - Click "Compile BootstrapAddressManager.sol".

5. **Deploy the Smart Contract:**
   - Click on the "Deploy & Run Transactions" tab (third icon from the top).
   - In the "Environment" dropdown, select "Injected Web3".
   - MetaMask will pop up asking for permission to connect to REMIX. Allow the connection.
   - Ensure you are connected to the Sepolia testnet in MetaMask.
   - Click "Deploy".

6. **Save the Contract Address:**
   - Once the contract is deployed, the contract address will appear in the "Deployed Contracts" section.
   - Copy the contract address for use in the botNode.
