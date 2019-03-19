package com.luxoft.fabric.impl;

import com.luxoft.fabric.FabricConnector;
import com.luxoft.fabric.FabricUser;
import org.hyperledger.fabric.sdk.Enrollment;
import org.hyperledger.fabric.sdk.NetworkConfig;
import org.hyperledger.fabric.sdk.User;
import org.hyperledger.fabric_ca.sdk.HFCAClient;
import org.hyperledger.fabric_ca.sdk.RegistrationRequest;
import org.hyperledger.fabric_ca.sdk.exception.InvalidArgumentException;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;


import java.net.MalformedURLException;


public class FabricConnectorImplBasedOnNetworkConfig extends FabricConnector {

    private static final Logger LOGGER = LoggerFactory.getLogger(FabricConnectorImplBasedOnFabricConfig.class);

    private final NetworkConfig networkConfig;

    private FabricConnectorImplBasedOnNetworkConfig(User user, String defaultChannelName, NetworkConfig networkConfig, Boolean initChannels, Options options) throws Exception {
        this.networkConfig = networkConfig;
        initConnector(user, defaultChannelName, initChannels, options);
    }

    @Override
    protected void initUserContext(User user) throws Exception {
        if (user != null)
            hfClient.setUserContext(user);
        else if (hfClient.getUserContext() == null)
            hfClient.setUserContext(networkConfig.getPeerAdmin());
    }

    @Override
    public void initChannels(Options options) throws Exception {
        for (String channelName : networkConfig.getChannelNames()) {
            hfClient.loadChannelFromConfig(channelName, networkConfig).initialize();
        }

    }

    @Override
    public User enrollUser(String caKey, String userName, String userSecret) throws Exception {
        HFCAClient hfcaClient = getHfcaClientForCaKey(caKey);
        Enrollment adminEnrollment = hfcaClient.enroll(userName, userSecret);
        return new FabricUser(userName, null, null, adminEnrollment, networkConfig.getClientOrganization().getMspId());
    }


    @Override
    public String registerUser(String caKey, String userName, String userAffiliation) throws Exception {

        NetworkConfig.CAInfo caInfo = getCAInfo(caKey);
        String adminUser = caInfo.getRegistrars().iterator().next().getName();
        String adminPassword = caInfo.getRegistrars().iterator().next().getEnrollSecret();

        LOGGER.debug("Retrieved adminUser: {}", adminUser);


        User admin = enrollUser(caKey, adminUser, adminPassword);
        RegistrationRequest registrationRequest = new RegistrationRequest(userName, userAffiliation);
        return getHfcaClientForCaKey(caKey).register(registrationRequest, admin);
    }

    private HFCAClient getHfcaClientForCaKey(String caKey) throws MalformedURLException, InvalidArgumentException {
        NetworkConfig.CAInfo caInfo = getCAInfo(caKey);
        return HFCAClient.createNewInstance(caInfo);
    }

    private NetworkConfig.CAInfo getCAInfo(String caKey) {
        for (NetworkConfig.CAInfo caInfo : networkConfig.getClientOrganization().getCertificateAuthorities()) {
            if (caInfo.getName().equals(caKey))
                return caInfo;
        }
        throw new IllegalArgumentException(String.format("No CA with name %s found", caKey));
    }

    public static class Builder extends FabricConnector.Builder{

        private NetworkConfig networkConfig;

        public Builder(NetworkConfig networkConfig) {

            this.networkConfig = networkConfig;
        }

        public FabricConnectorImplBasedOnNetworkConfig build() throws Exception {

            return new FabricConnectorImplBasedOnNetworkConfig(user, defaultChannelName, networkConfig, initChannels, options);
        }


    }

}
