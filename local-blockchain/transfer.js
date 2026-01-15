const { ethers } = require("ethers");

const RPC_URL = "http://127.0.0.1:32003"; 

const genesisAccounts = [
    { address: "0x8943545177806ED17B9F23F0a21ee5948eCaa776", pkey: "bcdf20249abf0ed6d944c0288fad489e33f66b3960d9e6229c1cd214ed3bbe31" },
    { address: "0xE25583099BA105D9ec0A67f5Ae86D90e50036425", pkey: "39725efee3fb28614de3bacaffe4cc4bd8c436257e2c8bb887c4b5c4be45e76d" },
];

const el1Account = "0xccdd5b657ab09ba1cc7e126e06664e9241836733";
const el2Account = "0x5e99D75F850f9118fCC711e64eD94E26f8F745f6";

async function main() {
    const provider = new ethers.JsonRpcProvider(RPC_URL);
    
    try {
        const network = await provider.getNetwork();
        console.log(`Conectado a la red: ${network.chainId})`);
    } catch (e) {
        console.error("Error conectando. Revisa el puerto RPC de Kurtosis.");
        return;
    }

    const whaleWallet = new ethers.Wallet(genesisAccounts[0].pkey, provider);

    const balanceBefore = await provider.getBalance(whaleWallet.address);
    console.log(`\nBalance de ${whaleWallet.address}:`);
    console.log(`${ethers.formatEther(balanceBefore)} ETH`);

    const amountToSend = ethers.parseEther("1.0"); 

    console.log(`\nEnviando 1.0 ETH a ${el2Account}...`);
    
    const tx = await whaleWallet.sendTransaction({
        to: el2Account,
        value: amountToSend
    });

    console.log(`Tx Hash: ${tx.hash}`);
    console.log("Esperando confirmación...");

    await tx.wait();
    console.log("Transacción confirmada en el bloque.");

    const balanceAfter = await provider.getBalance(whaleWallet.address);
    console.log(`\nNuevo Balance de ${whaleWallet.address}:`);
    console.log(`${ethers.formatEther(balanceAfter)} ETH`);
}

main().catch((error) => {
    console.error(error);
    process.exit(1);
});