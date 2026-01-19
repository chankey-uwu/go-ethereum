const hre = require("hardhat");

async function main() {
    const Storage = await hre.ethers.getContractFactory("Storage");

    const storage = await Storage.deploy();
    await storage.waitForDeployment();
    const address = await storage.getAddress();

    console.log("¡Contrato 'Storage' desplegado exitosamente!");
    console.log(`Dirección: ${address}`);
}

main().catch((error) => {
    console.error(error);
    process.exitCode = 1;
});