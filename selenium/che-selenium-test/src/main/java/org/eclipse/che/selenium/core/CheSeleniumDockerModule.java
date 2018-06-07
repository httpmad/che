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
package org.eclipse.che.selenium.core;

import com.google.inject.AbstractModule;
import org.eclipse.che.selenium.core.client.keycloak.executor.DockerContainerCommandExecutor;
import org.eclipse.che.selenium.core.client.keycloak.executor.KeycloakCommandExecutor;
import org.eclipse.che.selenium.core.workspace.CheTestDockerWorkspaceLogsReader;
import org.eclipse.che.selenium.core.workspace.TestWorkspaceLogsReader;

/** @author Dmytro Nochevnov */
public class CheSeleniumDockerModule extends AbstractModule {

  @Override
  protected void configure() {
    bind(TestWorkspaceLogsReader.class).to(CheTestDockerWorkspaceLogsReader.class);
    bind(KeycloakCommandExecutor.class).to(DockerContainerCommandExecutor.class);
  }
}
