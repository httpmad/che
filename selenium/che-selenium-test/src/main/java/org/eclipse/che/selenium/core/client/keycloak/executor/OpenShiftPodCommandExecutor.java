/*
 * Copyright (c) 2012-2018 Red Hat, Inc.
 * All rights reserved. This program and the accompanying materials
 * are made available under the terms of the Eclipse Public License v1.0
 * which accompanies this distribution, and is available at
 * http://www.eclipse.org/legal/epl-v10.html
 *
 * Contributors:
 *   Red Hat, Inc. - initial API and implementation
 */
package org.eclipse.che.selenium.core.client.keycloak.executor;

import static com.google.common.io.Files.createTempDir;
import static java.lang.String.format;

import com.google.inject.Inject;
import com.google.inject.Singleton;
import com.google.inject.name.Named;
import org.apache.commons.io.FileUtils;
import org.eclipse.che.selenium.core.provider.OpenShiftWebConsoleUrlProvider;
import org.eclipse.che.selenium.core.utils.process.ProcessAgent;
import org.eclipse.che.selenium.core.utils.process.ProcessAgentException;

import javax.annotation.PreDestroy;
import java.net.URL;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;

/** @author Dmytro Nochevnov */
@Singleton
public class OpenShiftPodCommandExecutor implements KeycloakCommandExecutor {

  private static final String DEFAULT_OPENSHIFT_USERNAME = "developer";
  private static final String DEFAULT_OPENSHIFT_PASSWORD = "any";

  private final static Path PATH_TO_OPEN_SHIFT_CLI_DIRECTORY = Paths.get(createTempDir().toString()).resolve("oc");
  private final static Path PATH_TO_OPEN_SHIFT_CLI = PATH_TO_OPEN_SHIFT_CLI_DIRECTORY.resolve("oc");
  public static final String ECLIPSE_CHE_NAMESPACE = "eclipse-che";

  private String keycloakPodName;

  @Inject
  private ProcessAgent processAgent;

  @Inject(optional = true)
  @Named("openshift.username")
  private String openShiftUsername;

  @Inject(optional = true)
  @Named("openshift.password")
  private String openShiftPassword;

  @Inject
  private OpenShiftWebConsoleUrlProvider openShiftWebConsoleUrlProvider;

  @Override
  public String execute(String command) throws ProcessAgentException {
    if (keycloakPodName == null || keycloakPodName.trim().isEmpty()) {
      obtainKeycloakPodName();
    }

    String openShiftCliCommand = format("%s exec %s -- /opt/jboss/keycloak/bin/kcadm.sh %s",
            PATH_TO_OPEN_SHIFT_CLI,
            keycloakPodName,
            command);

    return processAgent.process(openShiftCliCommand);
  }

  private void obtainKeycloakPodName() throws ProcessAgentException {
    if (Files.notExists(PATH_TO_OPEN_SHIFT_CLI)) {
      downloadOpenShiftCLI();
    }

    reLoginToOpenShift();

    // obtain name of keycloak pod
    keycloakPodName =
        processAgent.process(format("%s get pod --namespace=%s -l app=keycloak --no-headers | awk '{print $1}'",
                ECLIPSE_CHE_NAMESPACE,
                PATH_TO_OPEN_SHIFT_CLI));

    if (keycloakPodName.trim().isEmpty()) {
      throw new RuntimeException(
          format("Keycloak pod is not found at namespace %s at OpenShift instance %s.",
                  ECLIPSE_CHE_NAMESPACE,
                  openShiftWebConsoleUrlProvider.get()));
    }
  }

  private void downloadOpenShiftCLI() {

  }

  private void reLoginToOpenShift() throws ProcessAgentException {
    String reLoginToOpenShiftCliCommand = format("%1$s logout && %1$s login --server=%2$s -u=%3$s -p=%4$s --insecure-skip-tls-verify",
            PATH_TO_OPEN_SHIFT_CLI,
            openShiftWebConsoleUrlProvider.get(),
            openShiftUsername != null ? openShiftUsername : DEFAULT_OPENSHIFT_USERNAME,
            openShiftPassword != null ? openShiftPassword : DEFAULT_OPENSHIFT_PASSWORD);

    processAgent.process(reLoginToOpenShiftCliCommand);
  }

  @PreDestroy
  private void removeOpenShiftCli() {
    FileUtils.deleteQuietly(PATH_TO_OPEN_SHIFT_CLI_DIRECTORY.toFile());
  }

}
