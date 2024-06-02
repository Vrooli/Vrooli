export const runProjectOrRunRoutine_findMany = {
  "fieldName": "runProjectOrRunRoutines",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "runProjectOrRunRoutines",
        "loc": {
          "start": 1583,
          "end": 1606
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 1607,
              "end": 1612
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 1615,
                "end": 1620
              }
            },
            "loc": {
              "start": 1614,
              "end": 1620
            }
          },
          "loc": {
            "start": 1607,
            "end": 1620
          }
        }
      ],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "edges",
              "loc": {
                "start": 1628,
                "end": 1633
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "cursor",
                    "loc": {
                      "start": 1644,
                      "end": 1650
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1644,
                    "end": 1650
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "node",
                    "loc": {
                      "start": 1659,
                      "end": 1663
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "RunProject",
                            "loc": {
                              "start": 1685,
                              "end": 1695
                            }
                          },
                          "loc": {
                            "start": 1685,
                            "end": 1695
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "RunProject_list",
                                "loc": {
                                  "start": 1717,
                                  "end": 1732
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 1714,
                                "end": 1732
                              }
                            }
                          ],
                          "loc": {
                            "start": 1696,
                            "end": 1746
                          }
                        },
                        "loc": {
                          "start": 1678,
                          "end": 1746
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "RunRoutine",
                            "loc": {
                              "start": 1766,
                              "end": 1776
                            }
                          },
                          "loc": {
                            "start": 1766,
                            "end": 1776
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "RunRoutine_list",
                                "loc": {
                                  "start": 1798,
                                  "end": 1813
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 1795,
                                "end": 1813
                              }
                            }
                          ],
                          "loc": {
                            "start": 1777,
                            "end": 1827
                          }
                        },
                        "loc": {
                          "start": 1759,
                          "end": 1827
                        }
                      }
                    ],
                    "loc": {
                      "start": 1664,
                      "end": 1837
                    }
                  },
                  "loc": {
                    "start": 1659,
                    "end": 1837
                  }
                }
              ],
              "loc": {
                "start": 1634,
                "end": 1843
              }
            },
            "loc": {
              "start": 1628,
              "end": 1843
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "pageInfo",
              "loc": {
                "start": 1848,
                "end": 1856
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasNextPage",
                    "loc": {
                      "start": 1867,
                      "end": 1878
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1867,
                    "end": 1878
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorRunProject",
                    "loc": {
                      "start": 1887,
                      "end": 1906
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1887,
                    "end": 1906
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorRunRoutine",
                    "loc": {
                      "start": 1915,
                      "end": 1934
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1915,
                    "end": 1934
                  }
                }
              ],
              "loc": {
                "start": 1857,
                "end": 1940
              }
            },
            "loc": {
              "start": 1848,
              "end": 1940
            }
          }
        ],
        "loc": {
          "start": 1622,
          "end": 1944
        }
      },
      "loc": {
        "start": 1583,
        "end": 1944
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "projectVersion",
        "loc": {
          "start": 42,
          "end": 56
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 63,
                "end": 65
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 63,
              "end": 65
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "complexity",
              "loc": {
                "start": 70,
                "end": 80
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 70,
              "end": 80
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 85,
                "end": 93
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 85,
              "end": 93
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 98,
                "end": 107
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 98,
              "end": 107
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 112,
                "end": 124
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 112,
              "end": 124
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 129,
                "end": 141
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 129,
              "end": 141
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "root",
              "loc": {
                "start": 146,
                "end": 150
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 161,
                      "end": 163
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 161,
                    "end": 163
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 172,
                      "end": 181
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 172,
                    "end": 181
                  }
                }
              ],
              "loc": {
                "start": 151,
                "end": 187
              }
            },
            "loc": {
              "start": 146,
              "end": 187
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 192,
                "end": 204
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 215,
                      "end": 217
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 215,
                    "end": 217
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 226,
                      "end": 234
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 226,
                    "end": 234
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 243,
                      "end": 254
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 243,
                    "end": 254
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 263,
                      "end": 267
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 263,
                    "end": 267
                  }
                }
              ],
              "loc": {
                "start": 205,
                "end": 273
              }
            },
            "loc": {
              "start": 192,
              "end": 273
            }
          }
        ],
        "loc": {
          "start": 57,
          "end": 275
        }
      },
      "loc": {
        "start": 42,
        "end": 275
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 276,
          "end": 278
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 276,
        "end": 278
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 279,
          "end": 288
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 279,
        "end": 288
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "completedComplexity",
        "loc": {
          "start": 289,
          "end": 308
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 289,
        "end": 308
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "contextSwitches",
        "loc": {
          "start": 309,
          "end": 324
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 309,
        "end": 324
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startedAt",
        "loc": {
          "start": 325,
          "end": 334
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 325,
        "end": 334
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timeElapsed",
        "loc": {
          "start": 335,
          "end": 346
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 335,
        "end": 346
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "completedAt",
        "loc": {
          "start": 347,
          "end": 358
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 347,
        "end": 358
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 359,
          "end": 363
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 359,
        "end": 363
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "status",
        "loc": {
          "start": 364,
          "end": 370
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 364,
        "end": 370
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "stepsCount",
        "loc": {
          "start": 371,
          "end": 381
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 371,
        "end": 381
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "team",
        "loc": {
          "start": 382,
          "end": 386
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Team_nav",
              "loc": {
                "start": 396,
                "end": 404
              }
            },
            "directives": [],
            "loc": {
              "start": 393,
              "end": 404
            }
          }
        ],
        "loc": {
          "start": 387,
          "end": 406
        }
      },
      "loc": {
        "start": 382,
        "end": 406
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "user",
        "loc": {
          "start": 407,
          "end": 411
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "User_nav",
              "loc": {
                "start": 421,
                "end": 429
              }
            },
            "directives": [],
            "loc": {
              "start": 418,
              "end": 429
            }
          }
        ],
        "loc": {
          "start": 412,
          "end": 431
        }
      },
      "loc": {
        "start": 407,
        "end": 431
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 432,
          "end": 435
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 442,
                "end": 451
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 442,
              "end": 451
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 456,
                "end": 465
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 456,
              "end": 465
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 470,
                "end": 477
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 470,
              "end": 477
            }
          }
        ],
        "loc": {
          "start": 436,
          "end": 479
        }
      },
      "loc": {
        "start": 432,
        "end": 479
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "routineVersion",
        "loc": {
          "start": 523,
          "end": 537
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 544,
                "end": 546
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 544,
              "end": 546
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "complexity",
              "loc": {
                "start": 551,
                "end": 561
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 551,
              "end": 561
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isAutomatable",
              "loc": {
                "start": 566,
                "end": 579
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 566,
              "end": 579
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 584,
                "end": 594
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 584,
              "end": 594
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 599,
                "end": 608
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 599,
              "end": 608
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 613,
                "end": 621
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 613,
              "end": 621
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 626,
                "end": 635
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 626,
              "end": 635
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "root",
              "loc": {
                "start": 640,
                "end": 644
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 655,
                      "end": 657
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 655,
                    "end": 657
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isInternal",
                    "loc": {
                      "start": 666,
                      "end": 676
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 666,
                    "end": 676
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 685,
                      "end": 694
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 685,
                    "end": 694
                  }
                }
              ],
              "loc": {
                "start": 645,
                "end": 700
              }
            },
            "loc": {
              "start": 640,
              "end": 700
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 705,
                "end": 717
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 728,
                      "end": 730
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 728,
                    "end": 730
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 739,
                      "end": 747
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 739,
                    "end": 747
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 756,
                      "end": 767
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 756,
                    "end": 767
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "instructions",
                    "loc": {
                      "start": 776,
                      "end": 788
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 776,
                    "end": 788
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 797,
                      "end": 801
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 797,
                    "end": 801
                  }
                }
              ],
              "loc": {
                "start": 718,
                "end": 807
              }
            },
            "loc": {
              "start": 705,
              "end": 807
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 812,
                "end": 824
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 812,
              "end": 824
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 829,
                "end": 841
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 829,
              "end": 841
            }
          }
        ],
        "loc": {
          "start": 538,
          "end": 843
        }
      },
      "loc": {
        "start": 523,
        "end": 843
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 844,
          "end": 846
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 844,
        "end": 846
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 847,
          "end": 856
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 847,
        "end": 856
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "completedComplexity",
        "loc": {
          "start": 857,
          "end": 876
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 857,
        "end": 876
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "contextSwitches",
        "loc": {
          "start": 877,
          "end": 892
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 877,
        "end": 892
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startedAt",
        "loc": {
          "start": 893,
          "end": 902
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 893,
        "end": 902
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timeElapsed",
        "loc": {
          "start": 903,
          "end": 914
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 903,
        "end": 914
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "completedAt",
        "loc": {
          "start": 915,
          "end": 926
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 915,
        "end": 926
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 927,
          "end": 931
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 927,
        "end": 931
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "status",
        "loc": {
          "start": 932,
          "end": 938
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 932,
        "end": 938
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "stepsCount",
        "loc": {
          "start": 939,
          "end": 949
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 939,
        "end": 949
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "inputsCount",
        "loc": {
          "start": 950,
          "end": 961
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 950,
        "end": 961
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "wasRunAutomatically",
        "loc": {
          "start": 962,
          "end": 981
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 962,
        "end": 981
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "team",
        "loc": {
          "start": 982,
          "end": 986
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Team_nav",
              "loc": {
                "start": 996,
                "end": 1004
              }
            },
            "directives": [],
            "loc": {
              "start": 993,
              "end": 1004
            }
          }
        ],
        "loc": {
          "start": 987,
          "end": 1006
        }
      },
      "loc": {
        "start": 982,
        "end": 1006
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "user",
        "loc": {
          "start": 1007,
          "end": 1011
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "User_nav",
              "loc": {
                "start": 1021,
                "end": 1029
              }
            },
            "directives": [],
            "loc": {
              "start": 1018,
              "end": 1029
            }
          }
        ],
        "loc": {
          "start": 1012,
          "end": 1031
        }
      },
      "loc": {
        "start": 1007,
        "end": 1031
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1032,
          "end": 1035
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 1042,
                "end": 1051
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1042,
              "end": 1051
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1056,
                "end": 1065
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1056,
              "end": 1065
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1070,
                "end": 1077
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1070,
              "end": 1077
            }
          }
        ],
        "loc": {
          "start": 1036,
          "end": 1079
        }
      },
      "loc": {
        "start": 1032,
        "end": 1079
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1110,
          "end": 1112
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1110,
        "end": 1112
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 1113,
          "end": 1124
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1113,
        "end": 1124
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 1125,
          "end": 1131
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1125,
        "end": 1131
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 1132,
          "end": 1144
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1132,
        "end": 1144
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1145,
          "end": 1148
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canAddMembers",
              "loc": {
                "start": 1155,
                "end": 1168
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1155,
              "end": 1168
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 1173,
                "end": 1182
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1173,
              "end": 1182
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 1187,
                "end": 1198
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1187,
              "end": 1198
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 1203,
                "end": 1212
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1203,
              "end": 1212
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1217,
                "end": 1226
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1217,
              "end": 1226
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1231,
                "end": 1238
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1231,
              "end": 1238
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1243,
                "end": 1255
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1243,
              "end": 1255
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1260,
                "end": 1268
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1260,
              "end": 1268
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 1273,
                "end": 1287
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1298,
                      "end": 1300
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1298,
                    "end": 1300
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1309,
                      "end": 1319
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1309,
                    "end": 1319
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1328,
                      "end": 1338
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1328,
                    "end": 1338
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 1347,
                      "end": 1354
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1347,
                    "end": 1354
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 1363,
                      "end": 1374
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1363,
                    "end": 1374
                  }
                }
              ],
              "loc": {
                "start": 1288,
                "end": 1380
              }
            },
            "loc": {
              "start": 1273,
              "end": 1380
            }
          }
        ],
        "loc": {
          "start": 1149,
          "end": 1382
        }
      },
      "loc": {
        "start": 1145,
        "end": 1382
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1413,
          "end": 1415
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1413,
        "end": 1415
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1416,
          "end": 1426
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1416,
        "end": 1426
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1427,
          "end": 1437
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1427,
        "end": 1437
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 1438,
          "end": 1449
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1438,
        "end": 1449
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 1450,
          "end": 1456
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1450,
        "end": 1456
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 1457,
          "end": 1462
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1457,
        "end": 1462
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBotDepictingPerson",
        "loc": {
          "start": 1463,
          "end": 1483
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1463,
        "end": 1483
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 1484,
          "end": 1488
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1484,
        "end": 1488
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 1489,
          "end": 1501
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1489,
        "end": 1501
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "RunProject_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "RunProject_list",
        "loc": {
          "start": 10,
          "end": 25
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "RunProject",
          "loc": {
            "start": 29,
            "end": 39
          }
        },
        "loc": {
          "start": 29,
          "end": 39
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "projectVersion",
              "loc": {
                "start": 42,
                "end": 56
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 63,
                      "end": 65
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 63,
                    "end": 65
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "complexity",
                    "loc": {
                      "start": 70,
                      "end": 80
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 70,
                    "end": 80
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 85,
                      "end": 93
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 85,
                    "end": 93
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 98,
                      "end": 107
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 98,
                    "end": 107
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 112,
                      "end": 124
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 112,
                    "end": 124
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 129,
                      "end": 141
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 129,
                    "end": 141
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "root",
                    "loc": {
                      "start": 146,
                      "end": 150
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 161,
                            "end": 163
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 161,
                          "end": 163
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 172,
                            "end": 181
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 172,
                          "end": 181
                        }
                      }
                    ],
                    "loc": {
                      "start": 151,
                      "end": 187
                    }
                  },
                  "loc": {
                    "start": 146,
                    "end": 187
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 192,
                      "end": 204
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 215,
                            "end": 217
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 215,
                          "end": 217
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 226,
                            "end": 234
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 226,
                          "end": 234
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 243,
                            "end": 254
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 243,
                          "end": 254
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 263,
                            "end": 267
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 263,
                          "end": 267
                        }
                      }
                    ],
                    "loc": {
                      "start": 205,
                      "end": 273
                    }
                  },
                  "loc": {
                    "start": 192,
                    "end": 273
                  }
                }
              ],
              "loc": {
                "start": 57,
                "end": 275
              }
            },
            "loc": {
              "start": 42,
              "end": 275
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 276,
                "end": 278
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 276,
              "end": 278
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 279,
                "end": 288
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 279,
              "end": 288
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedComplexity",
              "loc": {
                "start": 289,
                "end": 308
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 289,
              "end": 308
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contextSwitches",
              "loc": {
                "start": 309,
                "end": 324
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 309,
              "end": 324
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startedAt",
              "loc": {
                "start": 325,
                "end": 334
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 325,
              "end": 334
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeElapsed",
              "loc": {
                "start": 335,
                "end": 346
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 335,
              "end": 346
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 347,
                "end": 358
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 347,
              "end": 358
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 359,
                "end": 363
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 359,
              "end": 363
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "status",
              "loc": {
                "start": 364,
                "end": 370
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 364,
              "end": 370
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stepsCount",
              "loc": {
                "start": 371,
                "end": 381
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 371,
              "end": 381
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "team",
              "loc": {
                "start": 382,
                "end": 386
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Team_nav",
                    "loc": {
                      "start": 396,
                      "end": 404
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 393,
                    "end": 404
                  }
                }
              ],
              "loc": {
                "start": 387,
                "end": 406
              }
            },
            "loc": {
              "start": 382,
              "end": 406
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "user",
              "loc": {
                "start": 407,
                "end": 411
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 421,
                      "end": 429
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 418,
                    "end": 429
                  }
                }
              ],
              "loc": {
                "start": 412,
                "end": 431
              }
            },
            "loc": {
              "start": 407,
              "end": 431
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 432,
                "end": 435
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 442,
                      "end": 451
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 442,
                    "end": 451
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 456,
                      "end": 465
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 456,
                    "end": 465
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 470,
                      "end": 477
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 470,
                    "end": 477
                  }
                }
              ],
              "loc": {
                "start": 436,
                "end": 479
              }
            },
            "loc": {
              "start": 432,
              "end": 479
            }
          }
        ],
        "loc": {
          "start": 40,
          "end": 481
        }
      },
      "loc": {
        "start": 1,
        "end": 481
      }
    },
    "RunRoutine_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "RunRoutine_list",
        "loc": {
          "start": 491,
          "end": 506
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "RunRoutine",
          "loc": {
            "start": 510,
            "end": 520
          }
        },
        "loc": {
          "start": 510,
          "end": 520
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "routineVersion",
              "loc": {
                "start": 523,
                "end": 537
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 544,
                      "end": 546
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 544,
                    "end": 546
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "complexity",
                    "loc": {
                      "start": 551,
                      "end": 561
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 551,
                    "end": 561
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 566,
                      "end": 579
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 566,
                    "end": 579
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 584,
                      "end": 594
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 584,
                    "end": 594
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 599,
                      "end": 608
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 599,
                    "end": 608
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 613,
                      "end": 621
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 613,
                    "end": 621
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 626,
                      "end": 635
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 626,
                    "end": 635
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "root",
                    "loc": {
                      "start": 640,
                      "end": 644
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 655,
                            "end": 657
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 655,
                          "end": 657
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isInternal",
                          "loc": {
                            "start": 666,
                            "end": 676
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 666,
                          "end": 676
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 685,
                            "end": 694
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 685,
                          "end": 694
                        }
                      }
                    ],
                    "loc": {
                      "start": 645,
                      "end": 700
                    }
                  },
                  "loc": {
                    "start": 640,
                    "end": 700
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 705,
                      "end": 717
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 728,
                            "end": 730
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 728,
                          "end": 730
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 739,
                            "end": 747
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 739,
                          "end": 747
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 756,
                            "end": 767
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 756,
                          "end": 767
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 776,
                            "end": 788
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 776,
                          "end": 788
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 797,
                            "end": 801
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 797,
                          "end": 801
                        }
                      }
                    ],
                    "loc": {
                      "start": 718,
                      "end": 807
                    }
                  },
                  "loc": {
                    "start": 705,
                    "end": 807
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 812,
                      "end": 824
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 812,
                    "end": 824
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 829,
                      "end": 841
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 829,
                    "end": 841
                  }
                }
              ],
              "loc": {
                "start": 538,
                "end": 843
              }
            },
            "loc": {
              "start": 523,
              "end": 843
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 844,
                "end": 846
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 844,
              "end": 846
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 847,
                "end": 856
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 847,
              "end": 856
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedComplexity",
              "loc": {
                "start": 857,
                "end": 876
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 857,
              "end": 876
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contextSwitches",
              "loc": {
                "start": 877,
                "end": 892
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 877,
              "end": 892
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startedAt",
              "loc": {
                "start": 893,
                "end": 902
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 893,
              "end": 902
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeElapsed",
              "loc": {
                "start": 903,
                "end": 914
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 903,
              "end": 914
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 915,
                "end": 926
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 915,
              "end": 926
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 927,
                "end": 931
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 927,
              "end": 931
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "status",
              "loc": {
                "start": 932,
                "end": 938
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 932,
              "end": 938
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stepsCount",
              "loc": {
                "start": 939,
                "end": 949
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 939,
              "end": 949
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 950,
                "end": 961
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 950,
              "end": 961
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "wasRunAutomatically",
              "loc": {
                "start": 962,
                "end": 981
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 962,
              "end": 981
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "team",
              "loc": {
                "start": 982,
                "end": 986
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Team_nav",
                    "loc": {
                      "start": 996,
                      "end": 1004
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 993,
                    "end": 1004
                  }
                }
              ],
              "loc": {
                "start": 987,
                "end": 1006
              }
            },
            "loc": {
              "start": 982,
              "end": 1006
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "user",
              "loc": {
                "start": 1007,
                "end": 1011
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 1021,
                      "end": 1029
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1018,
                    "end": 1029
                  }
                }
              ],
              "loc": {
                "start": 1012,
                "end": 1031
              }
            },
            "loc": {
              "start": 1007,
              "end": 1031
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1032,
                "end": 1035
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1042,
                      "end": 1051
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1042,
                    "end": 1051
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1056,
                      "end": 1065
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1056,
                    "end": 1065
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1070,
                      "end": 1077
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1070,
                    "end": 1077
                  }
                }
              ],
              "loc": {
                "start": 1036,
                "end": 1079
              }
            },
            "loc": {
              "start": 1032,
              "end": 1079
            }
          }
        ],
        "loc": {
          "start": 521,
          "end": 1081
        }
      },
      "loc": {
        "start": 482,
        "end": 1081
      }
    },
    "Team_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Team_nav",
        "loc": {
          "start": 1091,
          "end": 1099
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Team",
          "loc": {
            "start": 1103,
            "end": 1107
          }
        },
        "loc": {
          "start": 1103,
          "end": 1107
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1110,
                "end": 1112
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1110,
              "end": 1112
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 1113,
                "end": 1124
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1113,
              "end": 1124
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 1125,
                "end": 1131
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1125,
              "end": 1131
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 1132,
                "end": 1144
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1132,
              "end": 1144
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1145,
                "end": 1148
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canAddMembers",
                    "loc": {
                      "start": 1155,
                      "end": 1168
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1155,
                    "end": 1168
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1173,
                      "end": 1182
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1173,
                    "end": 1182
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1187,
                      "end": 1198
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1187,
                    "end": 1198
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 1203,
                      "end": 1212
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1203,
                    "end": 1212
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1217,
                      "end": 1226
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1217,
                    "end": 1226
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1231,
                      "end": 1238
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1231,
                    "end": 1238
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1243,
                      "end": 1255
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1243,
                    "end": 1255
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1260,
                      "end": 1268
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1260,
                    "end": 1268
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 1273,
                      "end": 1287
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1298,
                            "end": 1300
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1298,
                          "end": 1300
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 1309,
                            "end": 1319
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1309,
                          "end": 1319
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 1328,
                            "end": 1338
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1328,
                          "end": 1338
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 1347,
                            "end": 1354
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1347,
                          "end": 1354
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 1363,
                            "end": 1374
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1363,
                          "end": 1374
                        }
                      }
                    ],
                    "loc": {
                      "start": 1288,
                      "end": 1380
                    }
                  },
                  "loc": {
                    "start": 1273,
                    "end": 1380
                  }
                }
              ],
              "loc": {
                "start": 1149,
                "end": 1382
              }
            },
            "loc": {
              "start": 1145,
              "end": 1382
            }
          }
        ],
        "loc": {
          "start": 1108,
          "end": 1384
        }
      },
      "loc": {
        "start": 1082,
        "end": 1384
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 1394,
          "end": 1402
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 1406,
            "end": 1410
          }
        },
        "loc": {
          "start": 1406,
          "end": 1410
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1413,
                "end": 1415
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1413,
              "end": 1415
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1416,
                "end": 1426
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1416,
              "end": 1426
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1427,
                "end": 1437
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1427,
              "end": 1437
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 1438,
                "end": 1449
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1438,
              "end": 1449
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 1450,
                "end": 1456
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1450,
              "end": 1456
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 1457,
                "end": 1462
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1457,
              "end": 1462
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 1463,
                "end": 1483
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1463,
              "end": 1483
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1484,
                "end": 1488
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1484,
              "end": 1488
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 1489,
                "end": 1501
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1489,
              "end": 1501
            }
          }
        ],
        "loc": {
          "start": 1411,
          "end": 1503
        }
      },
      "loc": {
        "start": 1385,
        "end": 1503
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "runProjectOrRunRoutines",
      "loc": {
        "start": 1511,
        "end": 1534
      }
    },
    "variableDefinitions": [
      {
        "kind": "VariableDefinition",
        "variable": {
          "kind": "Variable",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 1536,
              "end": 1541
            }
          },
          "loc": {
            "start": 1535,
            "end": 1541
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "RunProjectOrRunRoutineSearchInput",
              "loc": {
                "start": 1543,
                "end": 1576
              }
            },
            "loc": {
              "start": 1543,
              "end": 1576
            }
          },
          "loc": {
            "start": 1543,
            "end": 1577
          }
        },
        "directives": [],
        "loc": {
          "start": 1535,
          "end": 1577
        }
      }
    ],
    "directives": [],
    "selectionSet": {
      "kind": "SelectionSet",
      "selections": [
        {
          "kind": "Field",
          "name": {
            "kind": "Name",
            "value": "runProjectOrRunRoutines",
            "loc": {
              "start": 1583,
              "end": 1606
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 1607,
                  "end": 1612
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 1615,
                    "end": 1620
                  }
                },
                "loc": {
                  "start": 1614,
                  "end": 1620
                }
              },
              "loc": {
                "start": 1607,
                "end": 1620
              }
            }
          ],
          "directives": [],
          "selectionSet": {
            "kind": "SelectionSet",
            "selections": [
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "edges",
                  "loc": {
                    "start": 1628,
                    "end": 1633
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "cursor",
                        "loc": {
                          "start": 1644,
                          "end": 1650
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 1644,
                        "end": 1650
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "node",
                        "loc": {
                          "start": 1659,
                          "end": 1663
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "selectionSet": {
                        "kind": "SelectionSet",
                        "selections": [
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "RunProject",
                                "loc": {
                                  "start": 1685,
                                  "end": 1695
                                }
                              },
                              "loc": {
                                "start": 1685,
                                "end": 1695
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "RunProject_list",
                                    "loc": {
                                      "start": 1717,
                                      "end": 1732
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 1714,
                                    "end": 1732
                                  }
                                }
                              ],
                              "loc": {
                                "start": 1696,
                                "end": 1746
                              }
                            },
                            "loc": {
                              "start": 1678,
                              "end": 1746
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "RunRoutine",
                                "loc": {
                                  "start": 1766,
                                  "end": 1776
                                }
                              },
                              "loc": {
                                "start": 1766,
                                "end": 1776
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "RunRoutine_list",
                                    "loc": {
                                      "start": 1798,
                                      "end": 1813
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 1795,
                                    "end": 1813
                                  }
                                }
                              ],
                              "loc": {
                                "start": 1777,
                                "end": 1827
                              }
                            },
                            "loc": {
                              "start": 1759,
                              "end": 1827
                            }
                          }
                        ],
                        "loc": {
                          "start": 1664,
                          "end": 1837
                        }
                      },
                      "loc": {
                        "start": 1659,
                        "end": 1837
                      }
                    }
                  ],
                  "loc": {
                    "start": 1634,
                    "end": 1843
                  }
                },
                "loc": {
                  "start": 1628,
                  "end": 1843
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "pageInfo",
                  "loc": {
                    "start": 1848,
                    "end": 1856
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "hasNextPage",
                        "loc": {
                          "start": 1867,
                          "end": 1878
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 1867,
                        "end": 1878
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorRunProject",
                        "loc": {
                          "start": 1887,
                          "end": 1906
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 1887,
                        "end": 1906
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorRunRoutine",
                        "loc": {
                          "start": 1915,
                          "end": 1934
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 1915,
                        "end": 1934
                      }
                    }
                  ],
                  "loc": {
                    "start": 1857,
                    "end": 1940
                  }
                },
                "loc": {
                  "start": 1848,
                  "end": 1940
                }
              }
            ],
            "loc": {
              "start": 1622,
              "end": 1944
            }
          },
          "loc": {
            "start": 1583,
            "end": 1944
          }
        }
      ],
      "loc": {
        "start": 1579,
        "end": 1946
      }
    },
    "loc": {
      "start": 1505,
      "end": 1946
    }
  },
  "variableValues": {},
  "path": {
    "key": "runProjectOrRunRoutine_findMany"
  }
} as const;
