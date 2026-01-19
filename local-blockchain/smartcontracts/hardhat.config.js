require("@nomicfoundation/hardhat-toolbox");

const PRIVATE_KEY = "bcdf20249abf0ed6d944c0288fad489e33f66b3960d9e6229c1cd214ed3bbe31";

module.exports = {
  solidity: "0.8.24",
  networks: {
    kurtosis: {
      url: "http://127.0.0.1:32003",
      chainId: 585858,
      accounts: [PRIVATE_KEY],
    },
  },
};