const { ethers } = require("ethers");
const fs = require("fs");
const path = require("path");

const RPC_URL = "http://127.0.0.1:32003";
const usbLocation = "E:/wallets/";

async function createAccountOnUSB() {
    const provider = new ethers.JsonRpcProvider(RPC_URL);
    
    try {
        const network = await provider.getNetwork();
        console.log(`Conectado a la red: ${network.chainId}`);
    } catch (e) {
        console.error("Error conectando. Revisa el puerto RPC de Kurtosis.");
        return;
    }

    const wallet = ethers.Wallet.createRandom();
    console.log("Nueva cuenta creada:");
    console.log(`Dirección: ${wallet.address}`);
    console.log(`Clave Privada: ${wallet.privateKey}`);

    const walletData = {
        address: wallet.address,
        privateKey: wallet.privateKey,
        publicKey: wallet.publicKey,
        mnemonic: wallet.mnemonic ? wallet.mnemonic.phrase : null,
        path: wallet.mnemonic ? wallet.mnemonic.path : null,
        createdAt: new Date().toISOString()
    };

    try {
        if (!fs.existsSync(usbLocation)) {
            fs.mkdirSync(usbLocation, { recursive: true });
            console.log(`Directorio creado: ${usbLocation}`);
        }

        const fileName = `${wallet.address}.json`;
        const filePath = path.join(usbLocation, fileName);

        fs.writeFileSync(filePath, JSON.stringify(walletData, null, 2));
        
        console.log(`Información guardada exitosamente en: ${filePath}`);

    } catch (error) {
        console.error("Error al guardar el archivo en la USB:");
        console.error(error.message);
    }
}

createAccountOnUSB();