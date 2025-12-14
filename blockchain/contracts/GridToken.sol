// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/token/ERC20/extensions/ERC20Burnable.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

contract GridToken is ERC20, ERC20Burnable, Ownable {
    constructor() ERC20("GridForce AI", "GRID") Ownable(msg.sender) {
        // İlk etapta 1 Milyon Token basıp kurucuya (sana) veriyoruz.
        _mint(msg.sender, 1000000 * 10 ** decimals());
    }

    // İleride Provider'lara ödül dağıtmak için yeni token basma yetkisi
    function mintReward(address to, uint256 amount) public onlyOwner {
        _mint(to, amount);
    }
}