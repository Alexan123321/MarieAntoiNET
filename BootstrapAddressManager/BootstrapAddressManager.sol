// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Define the contract named BootstrapAddressManager.
contract BootstrapAddressManager {
    // Array to store a list of bootstrap addresses privately within the contract.
    string[] private bootstrapAddresses;
    
    // Public state variable to store the owner's address of this contract.
    address public owner;

    // Events to be emitted on adding or removing an address. Events allow clients
    // (e.g., frontend applications) to react to contract state changes.
    event BootstrapAddressAdded(string indexed newAddress);
    event BootstrapAddressRemoved(string indexed removedAddress);

    // Modifier to restrict function access to only the owner of the contract.
    modifier onlyOwner() {
        require(msg.sender == owner, "Caller is not the owner");
        _; // Continue execution of the modified function.
    }

    // Constructor function that sets the initial owner of the contract to the address
    // that deployed the contract.
    constructor() {
        owner = msg.sender;
    }

    // Function to add a new bootstrap address. It's restricted to the contract's owner.
    function addBootstrapAddress(string memory _newAddress) public onlyOwner {
        bootstrapAddresses.push(_newAddress); // Add the new address to the array.
        emit BootstrapAddressAdded(_newAddress); // Emit an event for the new address addition.
    }

    // Public view function to retrieve all stored bootstrap addresses.
    function getBootstrapAddresses() public view returns (string[] memory) {
        return bootstrapAddresses;
    }

    // Function to remove a specific bootstrap address. It's also restricted to the contract's owner.
    function removeBootstrapAddress(string memory _addressToRemove) public onlyOwner {
        int256 index = findAddressIndex(_addressToRemove); // Find the index of the address to remove.
        require(index != -1, "Address not found"); // Ensure the address exists in the array.

        // Shift elements left to remove the address at the found index.
        for (uint i = uint(index); i < bootstrapAddresses.length - 1; i++) {
            bootstrapAddresses[i] = bootstrapAddresses[i + 1];
        }
        bootstrapAddresses.pop(); // Remove the last element now that the addresses have shifted.
        emit BootstrapAddressRemoved(_addressToRemove); // Emit an event for the address removal.
    }

    // Private view function to find the index of a given address in the array of bootstrap addresses.
    function findAddressIndex(string memory _address) private view returns (int256) {
        for (uint i = 0; i < bootstrapAddresses.length; i++) {
            // Check if the current address in the loop matches the provided address.
            if (keccak256(abi.encodePacked(bootstrapAddresses[i])) == keccak256(abi.encodePacked(_address))) {
                return int256(i); // Return the index if found.
            }
        }
        return -1; // Return -1 if the address is not found.
    }
}
