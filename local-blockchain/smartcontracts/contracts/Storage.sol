// contracts/Storage.sol
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract Storage {
    uint256 public number;
    
    event NumberChanged(uint256 timestamp, uint256 newNumber);

    function store(uint256 num) public {
        number = num;
        emit NumberChanged(block.timestamp, num);
    }

    function retrieve() public view returns (uint256) {
        return number;
    }
}