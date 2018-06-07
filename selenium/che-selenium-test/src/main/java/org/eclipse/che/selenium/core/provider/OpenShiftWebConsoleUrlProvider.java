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
package org.eclipse.che.selenium.core.provider;

import com.google.inject.Inject;
import com.google.inject.Provider;
import com.google.inject.Singleton;
import org.eclipse.che.selenium.core.utils.UrlUtil;

import javax.inject.Named;
import java.net.MalformedURLException;
import java.net.URL;
import java.util.regex.Pattern;

import static java.lang.String.format;

/** @author Dmytro Nochevnov */
@Singleton
public class OpenShiftWebConsoleUrlProvider implements Provider<URL> {

  private static final int PORT = 8443;
  private static final String PROTOCOL = "https";

  private static final String OPENSHIFT_HOST_REGEXP = ".*[\\.]([0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3})[.].*";

  @Inject
  @Named("che.host")
  private String cheHost;

  @Inject(optional = true)
  @Named("openshift.url")
  private String openShiftUrl;

  @Override
  public URL get() {
    if (openShiftUrl != null) {
      try {
        return new URL(openShiftUrl);
      } catch (MalformedURLException e) {
        throw new RuntimeException(e);
      }
    }

    String openShiftHost = extractOpenShiftHost();
    return UrlUtil.url(PROTOCOL, openShiftHost, PORT, "/");
  }

  private String extractOpenShiftHost(String cheHost) {
    if (openShiftUrl != null) {
      return openShiftUrl;
    }

    Pattern pattern = Pattern.compile(OPENSHIFT_HOST_REGEXP);
    if (!pattern.matcher(cheHost).matches()) {
      throw new RuntimeException(format("It's impossible to extract OpenShift host from Eclipse Che host '%s'. Make sure that correct value is set for `CHE_INFRASTRUCTURE`.", cheHost));
    }

    return pattern.matcher(cheHost).group(1);
  }
}
